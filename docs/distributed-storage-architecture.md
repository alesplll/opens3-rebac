# Распределённое хранение: проект архитектуры

Статус: **проект, не реализован**. Обновлено: 2026-09-11.

Текущий Storage — один локальный blob store. Этот документ фиксирует требования,
которые нужно решить до появления нескольких Storage nodes. Он не описывает
гарантии текущего runtime.

## Инварианты

1. Metadata публикует только committed version, для которой достигнута выбранная
   durability policy.
2. Независимые PUT создают разные logical versions. Retry одной внутренней
   операции узнаётся по operation ID, а не по ETag/content hash.
3. Logical blob имеет стабильный ID. Реплики либо принимают этот ID в Storage API,
   либо Metadata хранит mapping logical ID → node-local IDs.
4. Чтение использует только подтверждённые holders текущей version и умеет
   повторить запрос на другой реплике.
5. Topology/epoch является согласованным состоянием control plane; frontend может
   быть stateless только относительно локального диска.
6. Старый coordinator не может записать по устаревшей topology: Storage проверяет
   monotonic fencing token/epoch.

## N, W и R

`N` — число желаемых копий, `W` — число durable acknowledgements до commit, `R` —
число реплик, участвующих в read protocol.

Условие `W + R > N` даёт пересечение кворумов только при дополнительных
предпосылках: единый порядок versions, правильное чтение наиболее новой версии,
согласованная membership и обработка concurrent writes. Оно само по себе не
доказывает linearizability.

Для immutable blob с authoritative Metadata pointer возможна более простая схема:
запись получает W acknowledgements, Metadata сохраняет точный список подтвердивших
holders, чтение выбирает их и повторяется при ошибке. `N=3/W=2/R=1` безопасно
описывать только вместе с этим правилом; «читать с любой живой ноды» недостаточно.

## Идентификаторы и Storage API

Текущий StoreObject не принимает blob ID и создаёт UUID на каждой node. Поэтому
он не может без изменения контракта создать одинаковую реплику несколькими
вызовами.

Нужно выбрать один вариант:

- coordinator выдаёт logical blob ID, а Storage принимает его вместе с
  `operation_id` и `topology_epoch`;
- каждая node возвращает local blob ID, а Metadata атомарно хранит mapping всех
  подтверждённых реплик.

Первый вариант упрощает lookup/delete, второй сохраняет локальную автономность.
Решение должно быть принято до реализации Gateway fan-out.

## Data path

### Параллельная запись coordinator → Storage

Coordinator читает вход один раз, помещает chunks в bounded buffers и независимо
передаёт их N workers. Он ждёт W полных durable commits и отменяет/помечает
оставшиеся попытки для repair.

Обычный `io.MultiWriter` для этого недостаточен: он вызывает writers
последовательно, поэтому одна заблокированная pipe тормозит весь вход. Реализация
обязана определить backpressure, memory bound, client cancellation и поведение
после частичного успеха.

### Chain replication

Primary пересылает поток по цепочке, а acknowledgement возвращается после нужного
числа durable commits. Передачу можно pipeline-ить; задержка не равна автоматически
сумме полных времен записи каждой node. Схема требует изменения Storage, обработки
разрыва цепочки и fencing.

### Отдельный data proxy

Proxy скрывает topology от Gateway, но становится частью data path. Его frontend
можно масштабировать горизонтально, однако placement decisions, membership и
repair state требуют согласованного backend/control plane. GFS и Ceph полезны как
источники отдельных идей, но не являются буквальными примерами центрального
proxy: их клиенты передают data непосредственно storage servers/OSDs после
получения placement metadata.

Для первого прототипа рекомендуется параллельный coordinator с явным logical ID и
Metadata как authority. Это рекомендация проекта, а не утверждение о реализации.

## Placement и failure handling

Placement отвечает за membership, allocation, locate и repair. Его долговечное
состояние включает:

- текущий topology epoch;
- desired/confirmed holders;
- node health с таймстампами;
- очередь repair/rebalance и прогресс;
- tombstones до завершения удаления со всех поколений реплик.

Статус `suspect` исключает node из нового placement, но не является fencing.
Fencing требует токена, который Storage сравнивает с последним принятым epoch и
отклоняет старые записи.

Repair должен быть rate-limited и idempotent. При возвращении node checksums
помогают проверить bytes, но не определяют, какая topology/version актуальна.

## Delete и события

Событие удаления должно содержать logical resource/version ID и точные replica
references либо позволять Placement получить их из durable catalog. Очистка по
переиспользуемым bucket/key опасна: запоздавшее событие старого поколения способно
удалить данные нового объекта.

Delete marker в versioning-enabled bucket скрывает current object, но не удаляет
старые version blobs. Permanent version delete и physical GC — отдельные команды.

Kafka delivery проектируется как at-least-once. DB state и outbound event должны
фиксироваться через transactional outbox; consumers обязаны быть идемпотентными.

## Проверки до включения

- write succeeds только после W durable acknowledgements;
- timeout retry с тем же operation ID возвращает прежний terminal result;
- чтение не обращается к node, не подтвердившей current version;
- stale coordinator получает rejection по epoch;
- node loss не нарушает declared durability/read availability;
- repair не превышает заданный bandwidth/concurrency;
- late delete не затрагивает новое поколение resource;
- Metadata pointer не становится видимым до завершения write policy.

Источники для сравнения архитектур:
[The Google File System](https://static.googleusercontent.com/media/research.google.com/en/us/archive/gfs-sosp2003.pdf),
[Ceph architecture](https://docs.ceph.com/en/quincy/architecture/),
[Go io.MultiWriter](https://pkg.go.dev/io#MultiWriter).
