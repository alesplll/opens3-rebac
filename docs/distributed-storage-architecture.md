# Распределённое хранение — шардирование + репликация

Дата: 2026-04-06
Связанные документы: [Storage Service — план реализации](storage-service-implementation-plan.md) | [Kubernetes deployment](kubernetes-deployment-plan.md)

---

## Контекст

Сейчас Storage — один инстанс. Все blob лежат на одном диске. Это single point of failure и потолок по ёмкости. Этот документ описывает превращение Storage в **кластер нод**, где данные шардируются по consistent hash ring и реплицируются для отказоустойчивости.

**Ключевое свойство:** сам Storage-сервис **не меняется**. Каждая нода остаётся тупым blob-хранилищем. Вся координация — снаружи.

---

## Параметры кластера

```
N = replication factor (сколько копий каждого blob)
W = write quorum (сколько ACK ждать при записи)
R = read quorum (сколько нод опросить при чтении)

Гарантия консистентности: W + R > N
```

| Профиль | N | W | R | Свойство |
|---|---|---|---|---|
| Durability first | 3 | 3 | 1 | Запись медленная, чтение быстрое, максимальная надёжность |
| Balanced (quorum) | 3 | 2 | 2 | Стандартный quorum, strong consistency |
| **S3-like** | **3** | **2** | **1** | **Пишем в majority, читаем с любой живой — рекомендуемый** |
| Fast write | 3 | 1 | 3 | Запись быстрая, риск потери при крэше до репликации |

---

## Placement Service — новый сервис

Единственный новый компонент. Отвечает за:
- Какая нода хранит какие шарды (consistent hash ring)
- Какие ноды живы (heartbeat)
- Куда писать новый blob (Allocate)
- Где искать существующий blob (Locate)
- Фоновая дорепликация при потере ноды

```protobuf
service PlacementService {
  // Для записи: "на какие ноды положить этот blob?"
  rpc Allocate(AllocateRequest) returns (AllocateResponse);

  // Для чтения: "на каких нодах лежит этот blob?"
  rpc Locate(LocateRequest) returns (LocateResponse);

  // Storage-ноды репортят о себе
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

  // Административное: текущее состояние кластера
  rpc ClusterStatus(ClusterStatusRequest) returns (ClusterStatusResponse);
}
```

---

## Три альтернативных подхода к записи

### Альтернатива A: Gateway-driven parallel write

```
Gateway
  │
  ├── Placement.Allocate(blob_id, N=3) → [node-1, node-2, node-3]
  │
  ├──→ node-1.StoreObject(stream) ──→ ACK ─┐
  ├──→ node-2.StoreObject(stream) ──→ ACK ─┤ ждём W=2
  └──→ node-3.StoreObject(stream) ──→ ACK ─┘
  │
  └── Metadata.CreateObjectVersion(blob_id, nodes=[1,2,3])
```

Gateway мультиплексирует `io.Reader` на N потоков через `io.TeeReader` + `io.Pipe`:

```go
// Псевдокод в Gateway
readers := make([]io.Reader, N)
for i := range nodes {
    pr, pw := io.Pipe()
    readers[i] = pr
    go func() { storage[i].StoreObject(pr) }()
}
multiWriter := io.MultiWriter(pipeWriters...)
io.Copy(multiWriter, clientStream)  // один проход по данным
```

| | |
|---|---|
| **Плюсы** | Латентность = max(W самых быстрых нод). Storage не меняется вообще. Один проход по данным |
| **Минусы** | Gateway сильно усложняется (мультиплексирование стримов, обработка частичных ошибок). Gateway должен знать о топологии нод |
| **Подходит если** | Хотите минимум новых сервисов. Gateway уже достаточно сложный — ещё одна ответственность не критична |

### Альтернатива B: Chain replication (primary → replicas)

```
Gateway
  │
  ├── Placement.Allocate(blob_id, N=3) → [node-1 (primary), node-2, node-3]
  │
  └──→ node-1.StoreObject(stream)
          │
          ├──→ node-2.StoreObject(forward)
          │        │
          │        └──→ node-3.StoreObject(forward)
          │                    │
          │              ACK ──┘
          │        ACK ──┘
          ACK ──┘
```

Gateway пишет только на primary. Primary форвардит по цепочке.

| | |
|---|---|
| **Плюсы** | Gateway простой — пишет в одну ноду. Хорошо изучен (CRAQ, Chain Replication, HDFS Pipeline) |
| **Минусы** | Латентность = сумма всех хопов (последовательно). **Storage меняется** — нужна логика форвардинга. Если средняя нода в цепочке падает — цепочка рвётся |
| **Подходит если** | Хотите держать Gateway тонким. Готовы добавить логику форвардинга в Storage |

Что меняется в Storage при chain replication:

```go
// Новый метод или расширение StoreObject
type StorageService interface {
    // ... существующие методы ...

    // Список нод для форвардинга (пустой = конец цепочки)
    StoreObjectChain(ctx context.Context, reader io.Reader, size int64,
                     contentType string, forwardTo []NodeInfo) (*model.BlobMeta, error)
}
```

### Альтернатива C: Placement Service как прокси (самая чистая архитектура)

```
Gateway
  │
  └──→ Placement.StoreObject(stream, replication_factor=3)
          │
          │  (Placement сам решает куда, сам мультиплексирует)
          │
          ├──→ node-1.StoreObject(stream)
          ├──→ node-2.StoreObject(stream)
          └──→ node-3.StoreObject(stream)
          │
          ACK (blob_id, nodes)
```

Placement Service проксирует стримы. Gateway вообще не знает о количестве нод.

| | |
|---|---|
| **Плюсы** | Gateway остаётся максимально простым. Storage не меняется. Вся сложность в одном месте. Легко менять стратегию без трогания Gateway/Storage |
| **Минусы** | Placement на data-path — но он stateless, масштабируется горизонтально (как и сам Gateway). Основной trade-off: ещё один сервис для деплоя и мониторинга |
| **Подходит если** | Хотите чистое разделение ответственностей. Placement stateless за load balancer — масштабируется как Gateway |

---

## Сравнительная таблица

| Критерий | A: Gateway-driven | B: Chain replication | C: Placement-proxy |
|---|---|---|---|
| Изменения в Storage | Нет | Да (форвардинг) | Нет |
| Изменения в Gateway | Большие | Минимальные | Минимальные |
| Новые сервисы | Placement (лёгкий) | Placement (лёгкий) | Placement (тяжёлый) |
| Латентность записи | Лучшая (параллельно) | Худшая (последовательно) | Средняя (+1 hop) |
| Сложность реализации | Средняя | Средняя | Средняя (Placement stateless) |
| Data-path | Gateway | Storage chain | Placement (stateless, масштабируется) |
| Аналоги | Cassandra, DynamoDB | HDFS Pipeline, CRAQ | GFS, Ceph (OSD) |

### Рекомендация для учебного проекта

**Альтернатива A (Gateway-driven)** — лучший баланс:
- Storage не трогаем вообще
- Placement Service — лёгкий координатор (не data-path)
- Параллельная запись — лучшая латентность
- Gateway и так будет сложным — это его работа как оркестратора

---

## Consistent Hash Ring — как работает шардирование

```
         0 ──────── 90 ──────── 180 ──────── 270 ──────── 360
         │          │            │             │            │
      node-1     node-3       node-2        node-1       node-3
         │          │            │             │            │
         ▼          ▼            ▼             ▼            ▼
     ┌───────┐  ┌───────┐  ┌──────────┐  ┌───────┐  ┌───────┐
     │shard 0│  │shard 1│  │ shard 2  │  │shard 3│  │shard 4│
     └───────┘  └───────┘  └──────────┘  └───────┘  └───────┘

hash(blob_id) = 142 → попадает в shard 2 → primary: node-2
                                          → replicas: node-1, node-3 (следующие по кольцу)
```

При добавлении node-4 между node-2 и node-1 перемещается только ~1/4 данных (шарды, попавшие в зону node-4), а не все.

Библиотеки для Go:
- `github.com/serialx/hashring` — простой consistent hash
- `github.com/buraksezer/consistent` — bounded loads (более равномерное распределение)

---

## Изменения в других сервисах

### Metadata

Metadata должен хранить, на каких нодах лежат реплики:

```sql
-- Вариант 1: массив в JSONB (проще)
ALTER TABLE versions ADD COLUMN replica_nodes JSONB;
-- {"nodes": ["storage-1", "storage-2", "storage-3"]}

-- Вариант 2: отдельная таблица (нормализованно)
CREATE TABLE blob_replicas (
    version_id UUID REFERENCES versions(id) ON DELETE CASCADE,
    storage_node TEXT NOT NULL,
    blob_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'active', -- active, migrating, stale
    PRIMARY KEY (version_id, storage_node)
);
```

При чтении:
```
Gateway → Metadata.GetObjectMeta → blob_id + nodes
       → выбрать любую живую ноду (R=1)
       → Storage-N.RetrieveObject(blob_id)
```

### docker-compose

```yaml
storage-1:
  build: { context: ., dockerfile: services/storage/Dockerfile }
  ports: ["50053:50053"]
  volumes: [storage-data-1:/data]
  environment:
    GRPC_PORT: "50053"
    NODE_ID: "storage-1"

storage-2:
  build: { context: ., dockerfile: services/storage/Dockerfile }
  ports: ["50063:50063"]
  volumes: [storage-data-2:/data]
  environment:
    GRPC_PORT: "50063"
    NODE_ID: "storage-2"

storage-3:
  build: { context: ., dockerfile: services/storage/Dockerfile }
  ports: ["50073:50073"]
  volumes: [storage-data-3:/data]
  environment:
    GRPC_PORT: "50073"
    NODE_ID: "storage-3"

placement:
  build: { context: ., dockerfile: services/placement/Dockerfile }
  ports: ["50060:50060"]
  environment:
    STORAGE_NODES: "storage-1:50053,storage-2:50063,storage-3:50073"
    REPLICATION_FACTOR: "3"
    WRITE_QUORUM: "2"
    READ_QUORUM: "1"

volumes:
  storage-data-1:
  storage-data-2:
  storage-data-3:
```

---

## Kafka и репликация

При удалении объекта `object-deleted` должен доехать до **всех нод**, хранящих реплику:

**Вариант 1:** Каждая Storage-нода — свой consumer group → каждая получает сообщение:
```
consumer_group: storage-{node_id}-consumer
```
Минус: все ноды получают все сообщения, даже если blob у них нет. Нода делает `DeleteBlob` → файл не найден → идемпотентно ок.

**Вариант 2:** Placement Service потребляет `object-deleted` и рассылает `DeleteObject` gRPC только на нужные ноды:
```
Metadata → Kafka: object-deleted { blob_id }
Placement (consumer) → Locate(blob_id) → [node-1, node-3]
  → node-1.DeleteObject(blob_id)
  → node-3.DeleteObject(blob_id)
```
Минус: Placement на критическом пути удаления.

**Рекомендация:** Вариант 1 для простоты. Идемпотентность DeleteBlob делает лишние вызовы безвредными.

---

## Подводные камни

### 1. Split brain при network partition

Если Placement потеряет связь с нодой, но нода жива — Placement начнёт дорепликацию, а нода продолжит отвечать. Два источника правды.

**Решение:** Fencing — перед дорепликацией Placement ставит ноду в статус `suspect`. Gateway перестаёт писать на `suspect` ноды, но продолжает читать. Если нода вернётся — синхронизация по checksums.

### 2. Partial write при параллельной записи

Gateway пишет на 3 ноды. Две записали, третья упала. Blob с одним blob_id существует на 2 нодах из 3.

**Решение:** W=2 означает "достаточно". Gateway возвращает success. Placement фоново дореплицирует на третью ноду (или на replacement ноду).

### 3. Читаем stale данные после перезаписи

Object перезаписан (новый PutObject с тем же ключом). Metadata обновлён (новый blob_id). Но старый blob ещё не удалён с некоторых нод.

**Не проблема:** blob_id уникален (UUID). Новый PutObject создаёт **новый** blob_id. Старый blob_id → `object-deleted` через Kafka → eventually удалится.

### 4. Thundering herd при падении ноды

Если нода-1 упала и хранила 1000 blob — Placement пытается скопировать все 1000 одновременно на оставшиеся ноды, забивая сеть.

**Решение:** Rate-limited rebalance queue. Копировать по 10 blob одновременно, с паузой. Приоритизировать blob с `replica_count < N` (критически недореплицированные).

---

## Поэтапная реализация

```
Фаза 5a: Static sharding (MVP)
  │
  ├── Placement Service с фиксированным hash ring (конфиг из env)
  ├── 3 Storage-ноды в docker-compose
  ├── Gateway: Allocate → parallel write на N нод, ждать W=2
  ├── Gateway: Locate → читать с любой живой ноды (R=1)
  ├── Metadata: хранить replica_nodes
  └── Нет добавления/удаления нод в runtime
  │
Фаза 5b: Failure recovery
  │
  ├── Placement: heartbeat от нод (каждые 5s)
  ├── Placement: детекция упавшей ноды (3 пропущенных heartbeat)
  ├── Фоновая дорепликация: копирование blob с живой реплики на новую ноду
  ├── Gateway: при ошибке чтения → retry на другую реплику
  └── Тесты: убить ноду → данные доступны → дорепликация завершена
  │
Фаза 5c: Dynamic rebalance
  │
  ├── Добавление ноды: пересчёт hash ring → миграция ~1/N шардов
  ├── Dual-read: во время миграции читать и со старой, и с новой ноды
  ├── Throttling: фоновая миграция не убивает сеть (rate limit на байты/сек)
  └── Удаление ноды: перенести все шарды → вывести из ring
```

---

## Тесты

| Фаза | Тест | Что проверяет |
|---|---|---|
| 5a | Placement hash ring unit-тесты | Allocate/Locate возвращают правильные ноды |
| 5a | Интеграция: запись на N нод | blob доступен с любой реплики |
| 5b | Chaos: убить ноду | данные доступны с оставшихся реплик |
| 5b | Дорепликация | replica_count восстанавливается до N |
| 5c | Добавить ноду | шарды мигрируют, данные доступны во время миграции |
