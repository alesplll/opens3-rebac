# Распределённое хранение — шардирование и репликация

- Дата: 2026-04-06
- Актуализировано: 2026-09-11
Связанные документы: [Storage Service — план реализации](storage-service-implementation-plan.md) | [Kubernetes deployment](kubernetes-deployment-plan.md)

## Контекст и текущее состояние

Сейчас Storage работает как один локальный blob store: принимает поток, создаёт
`blob_id`, сохраняет файл на локальной файловой системе и возвращает его по этому
идентификатору. Placement Service, репликации и автоматического repair пока нет.

Цель этого плана — перейти к нескольким Storage nodes, сохранив простую модель:
Storage отвечает за байты, Metadata — за объектные версии, Placement — за
размещение и состояние реплик, Gateway или отдельный coordinator ведёт data path.

## Базовые параметры

- `N` — желаемое число реплик blob;
- `W` — число durable acknowledgements, после которых запись можно считать
  выполненной;
- `R` — число реплик, участвующих в чтении;
- `topology_epoch` — версия состава кластера и placement state;
- `operation_id` — идентификатор одной повторяемой команды записи.

Для первого прототипа можно исследовать `N=3`, `W=2`, `R=1`. Это ещё не готовая
гарантия strong consistency. Условие `W + R > N` полезно только вместе с единым
порядком версий, согласованным membership, выбором самой новой версии и обработкой
конкурентных записей.

В OpenS3 проще опираться на authoritative pointer в Metadata: после `W` durable
записей Metadata фиксирует конкретную committed version и список подтвердивших
holders. Чтение обращается только к ним и при ошибке пробует следующую реплику.

## Placement Service

Placement хранит и выдаёт:

- текущий `topology_epoch` и membership;
- назначенные и подтверждённые holders каждого logical blob;
- состояние nodes и время последнего health signal;
- очередь repair/rebalance;
- tombstones до завершения удаления со всех реплик.

Статус `suspect` запрещает назначать node для новых записей, но сам по себе не
останавливает старого coordinator. Для защиты от split brain Storage должен
сравнивать fencing token/epoch с последним принятым значением и отклонять stale
команды.

## Идентификатор реплики

Текущий `StoreObject` создаёт новый UUID внутри каждой Storage node. Поэтому
несколько независимых вызовов не создадут реплики с одинаковым `blob_id` без
изменения контракта. Возможны два варианта:

1. Coordinator выбирает logical `blob_id`, а Storage принимает его вместе с
   `operation_id` и `topology_epoch`.
2. Каждая node создаёт local `blob_id`, а Metadata/Placement хранит mapping
   `logical_blob_id → node + local_blob_id`.

Первый вариант проще для чтения и удаления. Второй меньше меняет автономность
Storage, но усложняет каталог реплик. Выбор нужен до реализации fan-out.

## Три варианта записи

### Вариант A: Gateway-driven parallel write

Gateway или выделенный coordinator получает targets от Placement, читает входной
поток один раз и передаёт chunks нескольким Storage workers через bounded buffers.
После `W` полных durable commits он регистрирует version в Metadata.

Плюсы: минимальное число новых компонентов и удобный первый прототип. Минусы:
Gateway знает topology, держит fan-out/backpressure и тратит исходящий bandwidth
на каждую реплику.

Обычный `io.MultiWriter` здесь недостаточен: writers вызываются последовательно,
поэтому одна медленная node блокирует весь поток. Нужны независимые workers,
ограничение памяти, cancellation и cleanup частичных записей.

### Вариант B: chain replication

Coordinator отправляет поток primary node, та пересылает его дальше по цепочке.
Acknowledgement возвращается после заданного числа durable commits. Передачу
можно pipeline-ить, поэтому задержка не обязана равняться сумме полных времён
записи на каждой node.

Плюсы: Gateway отправляет данные один раз. Минусы: Storage становится участником
протокола репликации; нужны reconfiguration, fencing и обработка разрыва цепочки.

### Вариант C: Placement/Data Proxy

Отдельный proxy получает placement и ведёт fan-out, а Gateway видит один data
endpoint. Frontend proxy можно масштабировать горизонтально, но membership,
epochs и repair state должны храниться в согласованном control plane.

Плюсы: Gateway не знает topology, data path можно развивать отдельно. Минусы:
появляется ещё один нагруженный компонент и сетевой hop.

GFS и Ceph полезны как источники идей о placement и recovery, но не являются
точными примерами такого proxy: после получения metadata их data path идёт к
storage servers/OSDs.

## Сравнение вариантов

| Критерий | A: parallel coordinator | B: chain | C: data proxy |
|---|---|---|---|
| Изменения Storage API | logical ID/epoch | протокол репликации | logical ID/epoch |
| Нагрузка на Gateway | высокая | ниже | низкая |
| Новый компонент | не обязателен | не обязателен | обязателен |
| Сложность failure handling | средняя | высокая | высокая |
| Подход для первого этапа | **да** | позже | после измерений |

Рекомендуемый первый этап — вариант A с coordinator-selected logical ID и
Metadata как источником истины. Это решение для прототипа, которое нужно
подтвердить тестами; варианты B и C остаются кандидатами при росте нагрузки.

## Consistent hash ring

Ring может выбирать предпочтительные nodes для нового blob:

1. Placement хэширует logical `blob_id`.
2. Находит следующую virtual node на ring.
3. Выбирает `N` разных физических nodes с учётом failure domain.
4. Сохраняет назначение и затем отдельно отмечает durable acknowledgements.

Ring не заменяет каталог фактических holders. После падения, частичной записи или
rebalance вычисленное размещение может отличаться от реально подтверждённого.
Чтение использует committed catalog, а ring — для новых назначений и repair.

## Изменения в сервисах

### Gateway/coordinator

- получает placement и fencing epoch;
- передаёт один поток репликам с bounded backpressure;
- ждёт `W` durable результатов;
- повторяет ту же команду по `operation_id`;
- после частичного успеха создаёт durable cleanup/repair work.

### Storage

- принимает выбранную схему logical/local ID;
- проверяет `topology_epoch`;
- идемпотентно возвращает terminal result одной операции;
- сообщает checksum и durable commit;
- поддерживает repair copy и удаление конкретной реплики.

### Metadata

- делает version видимой только после выполнения write policy;
- хранит logical blob и подтверждённых holders либо ссылку на Placement catalog;
- не использует ETag как ключ идемпотентности;
- публикует cleanup/repair events через transactional outbox.

## Kafka, delete и repair

Kafka подходит для repair, rebalance и cleanup, которым не требуется держать
клиентский запрос открытым. Доставка проектируется как at-least-once: события
имеют `event_id`, resource generation, version и точные replica references, а
consumers выполняют команды идемпотентно.

Удаление по одному bucket/key опасно: позднее событие старого поколения может
затронуть новый объект с тем же именем. Delete marker скрывает текущий объект, но
не удаляет старые version blobs. Permanent version delete и physical GC идут
отдельным flow.

Repair ограничивается по bandwidth/concurrency. Checksums подтверждают байты, но
актуальную version и topology определяет control plane.

## Основные failure modes

1. **Split brain.** Старый coordinator пишет после изменения topology — Storage
   отклоняет stale epoch.
2. **Partial write.** Записано меньше `W` или ответ потерян — операция остаётся
   незавершённой, а подтверждённые orphan replicas попадают в cleanup/repair.
3. **Stale read.** Клиент попал на node без current version — чтение выбирает
   holders из committed Metadata и умеет сделать retry.
4. **Thundering herd.** После падения node repair ограничивается очередью,
   приоритетами и лимитами, а не стартует для всех blob одновременно.
5. **Late delete.** Команда содержит поколение/version/blob IDs и не удаляет
   ресурс только по переиспользуемому имени.

## Поэтапная реализация

1. Зафиксировать logical ID, operation ID и fencing в protobuf.
2. Реализовать Placement membership и durable replica catalog.
3. Добавить parallel coordinator для `N=3/W=2`.
4. Перевести read path на confirmed holders с retry.
5. Добавить outbox, cleanup и rate-limited repair.
6. Провести node-loss, partition и rebalance tests.
7. По метрикам решить, нужен ли chain replication или отдельный data proxy.

## Проверки

- success записи возможен только после `W` durable acknowledgements;
- retry с тем же `operation_id` возвращает тот же terminal result;
- независимый PUT одинаковых байтов создаёт новую logical version;
- чтение не выбирает неподтвердившую current version node;
- stale coordinator получает rejection;
- падение одной node соответствует заявленной durability/availability;
- repair не превышает заданные лимиты;
- late delete не затрагивает новое поколение ресурса;
- Metadata pointer не виден до выполнения write policy.

Материалы для сравнения:
[The Google File System](https://static.googleusercontent.com/media/research.google.com/en/us/archive/gfs-sosp2003.pdf),
[Ceph architecture](https://docs.ceph.com/en/quincy/architecture/),
[Go io.MultiWriter](https://pkg.go.dev/io#MultiWriter).
