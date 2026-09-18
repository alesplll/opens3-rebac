# Gateway Service

Минимальный HTTP-вход в локальный стенд OpenS3-ReBAC. Это первый PUT/GET-срез,
а не полный S3 API.

## Назначение и возможности

Gateway принимает обычный PUT/GET объекта, передаёт байты в Storage и работает с
каталогом Metadata. HTTP-сервер также предоставляет `GET /healthz` и завершает
активные запросы при SIGINT/SIGTERM.

## Архитектура и зависимости

```text
HTTP client → Gateway → Metadata gRPC (bucket и версии)
                      → Storage gRPC (байты blob)
```

Для работы с объектами нужны доступные Metadata и один Storage node. Kafka и
PostgreSQL остаются зависимостями Metadata, но Gateway к ним напрямую не
подключается. Внутренние gRPC-соединения в локальном стенде используют plaintext.

## Структура проекта

```text
cmd/server/                 запуск процесса и обработка сигналов
internal/config/            адреса HTTP и gRPC
internal/app/               соединения и жизненный цикл HTTP-сервера
internal/handler/httpapi/   PUT, GET и health endpoint
```

## API

| Метод и путь | Результат |
|---|---|
| `PUT /{bucket}/{key}` | сохранить тело и зарегистрировать текущую версию; `200`, `ETag`, `X-Object-Version-Id` |
| `GET /{bucket}/{key}` | вернуть байты текущей версии, `Content-Type`, `Content-Length`, `ETag` |
| `GET /healthz` | `200 ok` при работающем HTTP-процессе |

`key` может содержать `/`. Пустые bucket/key не принимаются. GET по `version_id`,
HTTP Range, HEAD, DELETE, multipart и S3 authentication пока отсутствуют. Health
проверяет сам процесс, а не готовность Metadata и Storage.

## Данные и основные сценарии

PUT проверяет существование bucket через Metadata `HeadBucket`, отправляет тело
потоком в Storage `StoreObject`, затем вызывает Metadata `CreateObjectVersion` с
полученным `blob_id`, размером и MD5. HTTP-успех возвращается после ответа
Metadata. GET получает текущий `blob_id` через `GetObjectMeta` и передаёт ответ
Storage `RetrieveObject` клиенту потоком.

Этот путь использует существующий `CreateObjectVersion`, который сразу создаёт
committed-версию. `pending/committed/aborted` orchestration, operation ID и
восстановление после разрыва между записью blob и Metadata ещё не реализованы.
При ошибке Metadata после успешной записи в Storage может остаться blob без
ссылки в каталоге.

## Конфигурация

| Переменная | Локальное значение по умолчанию | Compose |
|---|---|---|
| `GATEWAY_HTTP_ADDR` | `127.0.0.1:8080` | `:8080` |
| `METADATA_GRPC_ADDR` | `localhost:50052` | `metadata:50052` |
| `STORAGE_GRPC_ADDR` | `localhost:50053` | `storage:50053` |

Compose публикует HTTP-порт только на `127.0.0.1:8080`, поскольку в этом срезе
пока нет клиентской аутентификации и авторизации.

## Запуск

Из корня репозитория:

```bash
make up-gateway-mvp
docker compose --profile services ps gateway metadata storage
```

Команда поднимает Gateway, Metadata, Storage и инфраструктурные зависимости
Metadata. Bucket должен быть создан заранее через Metadata. Для запуска вне
Docker поднимите Metadata и Storage, затем:

```bash
cd services/gateway
go run ./cmd/server
```

## Примеры использования

Создайте тестовый bucket через существующий Metadata RPC (нужен `grpcurl`):

```bash
grpcurl -plaintext -d '{"name":"gateway-demo","owner_id":"11111111-1111-4111-8111-111111111111"}' \
  localhost:50052 opens3.metadata.v1.MetadataService/CreateBucket
```

После этого:

```bash
printf 'hello' | curl -i -X PUT -H 'Content-Type: text/plain' --data-binary @- \
  http://127.0.0.1:8080/gateway-demo/example.txt
curl -i http://127.0.0.1:8080/gateway-demo/example.txt
curl -i http://127.0.0.1:8080/healthz
```

## Наблюдаемость и диагностика

Ошибки downstream gRPC отображаются в HTTP 400/404/503/507/504 или 502. При
ошибке регистрации уже записанного blob Gateway пишет `blob_id` в лог для
ручной диагностики. Для логов контейнера: `docker compose logs -f gateway`.

## Разработка и тесты

```bash
make test-gateway
docker compose --profile services config --quiet
```

Проверка полного пути требует поднятых Metadata и Storage и тестового bucket.
Изменение protobuf для этого первого среза не требуется.

## Ограничения и дальнейшие работы

Этот локальный HTTP endpoint пока не проверяет пользователя и права, не
резервирует Quota, не поддерживает S3 SigV4 и не обещает атомарность Storage с
Metadata. Следующий шаг для жизненного цикла версии: `operation_id`, создание
pending-версии и отдельная фиксация в Metadata. После этого можно добавить
восстановление зависших операций.

## Связанные документы

- [Metadata](../metadata/README.md) и [Storage](../storage/README.md).
- [Проект политики Gateway](../../docs/gateway-contract-plan.md).
- [Первый запуск](../../GETTING_STARTED.md).
