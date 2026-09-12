# Gateway: проект политики доступа и событий

Статус на 11 сентября 2026: **предложение для реализации**, Gateway отсутствует.
Этот документ восстанавливает межсервисные таблицы, потерянные при сокращении
CLAUDE.md. Он не означает, что политика уже утверждена владельцем проекта или
реализована. Открытое решение D12 нельзя закрыть одной заменой resource prefix.

## Границы ответственности

- Gateway аутентифицирует клиента, выводит identity и управляет запросом.
- Auth проверяет credentials через Users и выпускает JWT; ValidateToken возвращает
  Empty, поэтому передачу подтверждённой identity ещё нужно согласовать.
- AuthZ принимает subject/action/resource, но не проверяет полномочия вызывающего
  WriteTuple/DeleteTuple. Gateway обязан защищать операции изменения графа.
- Quota резервирует расход до записи; повторное положительное начисление запрещено.
- Storage фиксирует байты, Metadata — видимую версию; общий success barrier
  проходит после обоих шагов.

Bearer JWT сам по себе не реализует внешний SigV4-контракт. Выбор собственного
HTTP API либо S3 authentication должен быть отдельным решением Gateway.

## Предлагаемый mapping операций

Все subject берутся из подтверждённой identity, а не из произвольного тела запроса.
Таблица — единое предложение; при принятии policy обновляются код, тесты и схемы.

| Операция | Ресурс | Предлагаемая проверка | Дополнительное условие |
|---|---|---|---|
| GetObject / HeadObject | `object:<bucket>/<key>` | READ | читать конкретную committed version |
| ListObjects / HeadBucket | `bucket:<name>` | READ | не подменять object-проверкой |
| Новый PutObject | `bucket:<name>` | CREATE | ресурс нового object ещё не существует |
| Overwrite | `object:<bucket>/<key>` | WRITE | смена current pointer после Storage commit |
| CreateBucket | ещё нет bucket | authenticated account policy | reserve bucket quota; без Check на вымышленном ресурсе |
| ListBuckets | учётная запись | authenticated identity | owner_id из identity, не произвольный request |
| DeleteObject | `object:<bucket>/<key>` | DELETE | отдельно выбрать unversioned/delete marker/permanent version delete |
| DeleteBucket | `bucket:<name>` | ADMIN | lifecycle emptiness и запрет гонки с новыми записями |
| Initiate multipart | bucket при новом key, object при overwrite | CREATE / WRITE соответственно | durable связь upload с key и initiator |
| UploadPart / Complete | ресурс, сохранённый в upload intent | повторная CREATE / WRITE по типу загрузки | проверить initiator и допустимое состояние upload |
| Abort multipart | ресурс из upload intent | initiator с правами на загрузку либо ADMIN | не отменять committed результат |
| Grant / Revoke direct permission | точный target resource | ADMIN у инициатора | затем WriteTuple / DeleteTuple |

Policy членства в группах требует отдельного решения об администраторах группы:
ADMIN на bucket не даёт автоматически право менять MEMBER_OF произвольной группы.

Уровни включают нижние: ADMIN > DELETE > CREATE > WRITE > READ. Следовательно,
CREATE выше WRITE; если нужны независимые полномочия, потребуется изменение модели.
Само ребро PARENT_OF не распространяет bucket permission на новый object. После
создания нужно либо явно назначать direct tuples по принятой policy, либо отдельно
реализовать наследование. Приведённый mapping не решает это автоматически.

## Write flow и неопределённый исход

```text
Аутентификация → AuthZ Check → CheckQuota(+delta)
  → Storage StoreObject/Complete → Metadata CreateObjectVersion
  → видимая committed version → внешний успех
```

При отказе до reserve ничего не компенсируется. При подтверждённом abort после
reserve выполняется compensation. При timeout исход может быть неизвестен:
требуется durable operation ID/terminal result и reconciliation, а не слепой
retry или немедленная отрицательная дельта. Квота bytes/objects при overwrite и
version retention должна учитывать выбранную модель начисления, а не всегда `+1`.

Для pending/finalize альтернативы сначала сохраняется durable intent. Затем
Storage подтверждает commit, Metadata атомарно меняет current pointer, и только
после этого Gateway возвращает успех. Event-driven вариант может ждать correlated
terminal event вместо синхронного ответа; требование внешней видимости сохраняется.

## Существующие события

| Topic | Producer | Consumer | Payload | Retry / ограничение |
|---|---|---|---|---|
| `auth-changes` | AuthZ AuditProducer | отдельный cache invalidator для tuple events | tuple_written/tuple_removed: timestamp, tuple(subject/relation/object), invalidation_hints | hints не покрывают все group-derived decisions; нет outbox |
| `auth-changes` | тот же producer | audit sink не подключён как гарантия | ACCESS_GRANTED/ACCESS_DENIED: timestamp, subject, action, object | отдельного allowed/from_cache/level нет; callback логирует failure |
| `object-deleted` | Metadata | Storage/AuthZ cleanup пока отсутствуют | object_id, blob_id текущей версии | publish после DELETE, повтор не гарантирует восстановление event |
| `bucket-deleted` | Metadata | AuthZ cleanup пока отсутствует | bucket_id, bucket_name | publish после commit, outbox отсутствует |

Имена соответствуют development wiring; Metadata topics читаются из config,
AuthZ topic сейчас задан default конструктора. Внутри Docker Kafka — `kafka:29092`,
с хоста — `localhost:9092`.

## Проектируемые события

Названия ниже — кандидаты, **не существующая интеграция**:

| Кандидат topic | Producer → Consumer | Минимальный смысл payload | Идемпотентность |
|---|---|---|---|
| `object-stored` | Storage/coordinator → Metadata recovery | operation_id, version_id, blob_id, size, checksum, durable intent correlation | finalize только известного intent; одинаковый terminal result |
| `object-aborted` | lifecycle coordinator → Storage cleanup | event_id, operation_id, upload_id/generation, точные pending blob refs | не удалять committed version или upload другого поколения |
| `object-deleted` (расширение) | Metadata outbox → Storage GC/AuthZ | event_id, generation, version_id, полный набор blob/replica refs | отдельные durable dedup/checkpoints каждого consumer |
| `bucket-deleted` (расширение) | Metadata outbox → AuthZ cleanup | event_id, bucket_id/generation, bucket_name | позднее событие не затрагивает новый bucket с тем же именем |

Исторические `object-blob-stored` / `object-delete-requested` — альтернативные
названия, не дополнительные обязательные topics. Event schemas нужно принять до
реализации producer и consumer. Payload одного blob не позволяет восстановить
bucket/key/owner без durable intent. Delete marker не является командой GC всех
исторических версий.

## Ошибки внешнего API

Единого готового HTTP mapping пока нет. Gateway должен учитывать операцию и
domain error, а не механически переводить любой gRPC код в один HTTP статус:
missing bucket, missing key, invalid range, quota denial и временная недоступность
зависимости имеют разный смысл. `507/QuotaExceeded` может быть расширением проекта,
но его нельзя объявлять универсальным S3-контрактом.

Для будущих S3 XML errors отдельно фиксируются Code, Message, Resource и RequestId;
примеры внутренних gRPC errors находятся в README сервисов. `UNAVAILABLE` не
доказывает, что мутация не произошла, и не делает её повтор безопасным.

## Критерии принятия

- Неверная identity или DENY не приводят к mutation или reserve.
- Создание нового key, overwrite и multipart проходят одну согласованную policy.
- Grant/revoke защищены проверкой инициатора; race/stale cache учитываются явно.
- Потерянный ответ не создаёт второй reserve/version и не освобождает успешный reserve.
- Следующий GET/HEAD/LIST после внешнего успеха видит committed version.
- Late delete адресует поколение/версию, а не только переиспользуемое имя.
