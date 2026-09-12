# Фактологический аудит Markdown, GitHub Wiki и issues

Дата проверки: 10 сентября 2026 года. Статус: **выполнено автономно; см. отметки по каждому пункту**.

Проверены 23 Markdown-файла исходного checkout, все 18 Markdown-страниц опубликованной Wiki и тексты всех 37 issues, включая закрытые. После проверки один устаревший файл удалён по решению владельца. Повторяющиеся проблемы объединены по исправлению; это список обнаруженных проблем, а не утверждение об отсутствии любых других дефектов.

Основания: исходный код, protobuf, миграции, Makefile, Compose, CI и первичная документация внешних систем. Исходный checkout: `e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75`; исходная Wiki: `009dbd2905ef190dc4f6f16ec93801870ebabf9d`, исправленная Wiki: `db3a1c7`. Неприменённые изменения пользователя в `services/authz/proto/generate.sh` и `.DS_Store` сохранены и не входят в PR.

Полный кластер и DB integration tests не запускались: Docker daemon недоступен. Unit tests Metadata прошли; integration-tag test с исправленным repository скомпилирован без запуска БД. Rust toolchain в среде отсутствует, поэтому Quota regression test проверен статически, но не запущен. Внешние картинки/Excalidraw не входили в проверку текста; Mermaid и текстовые схемы проверялись. Три отдельных checkout в `.claude/worktrees/` исключены как другие рабочие деревья. 16 страниц локальной копии `opens3-rebac.wiki/` сверены с опубликованной Wiki; исправления внесены во все 18 опубликованных страниц.

## Повторная проверка 11 сентября 2026

Отметки ниже отражают первый проход исправлений, а не окончательное подтверждение
всех примеров и runtime-сценариев. [Повторная проверка](documentation-review-2026-09-11.md)
обнаружила оставшиеся ошибки в README/схемах и потери полезных инструкций;
исправления и восстановленное содержание внесены в эту ветку. Для D12 пока
подготовлено предложение policy; его принятие не подтверждено. Проверки C01/C04
не покрывают все изначально запрошенные concurrent/fail-retry сценарии.

## Статусы выполнения

Всего 82 пункта: 32 по документации проекта, 19 по Wiki/архитектуре, 26 по issues и 5 по реализации. Это не 82 независимых дефекта: некоторые пункты уточняют один риск в разных контрактах. `✅` означает выполненную правку, `📌` — корректно сформулированную отдельную задачу, `👀`/`⚠️` — решение владельца. Закрытые исторические issues не переоткрывались и не переписывались так, будто новая реализация существовала в прошлом.

Приоритеты: **P1** — риск неверных прав, потери/невидимости данных, неправильной реализации API; **P2** — неверный контракт, запуск или существенное техническое утверждение; **P3** — устаревшая навигация, неподтверждённые оценки, редакционная неточность. Метки «текущее состояние», «план» и «исторический снимок» должны быть явно различимы.

## Документы проекта

### D01 · P2 · Состав сервисов и технологический стек устарели

**Статус: ✅ исправлено в PR.**

**Где:** [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md), [GETTING_STARTED.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/GETTING_STARTED.md), [AGENTS.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/AGENTS.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md); Wiki Home/Services/Architecture.

Встречаются четыре реализованных сервиса, отсутствующий Metadata и Python для Metadata. В checkout есть Auth, Users, AuthZ, Metadata, Storage и Quota; Metadata написан на Go, Quota — на Rust. Gateway пока отсутствует. Версии Go/Rust на корневых бейджах расходятся с `go.mod` и Dockerfile.

**Исправить:** дать таблицу «реализован / планируется», Go 1.24.1 и фактический Rust toolchain, добавить Metadata/Quota в обзор и инструкции для разработчиков. Основание: [go.work](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/go.work), [services/metadata/go.mod](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/go.mod), [services/quota/Dockerfile](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/Dockerfile).

### D02 · P1 · Планируемая S3-совместимость и репликация представлены как готовые

**Статус: ✅ исправлено в PR.**

**Где:** [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md), [docs/class_diagram.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/class_diagram.md), [docs/object_diagram.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/object_diagram.md), [docs/sequence_diagrams.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/sequence_diagrams.md); Wiki Home/Architecture/Services.

Текст обещает работу стандартных S3-клиентов и описывает Gateway, WAL, placement, репликацию и некоторые Kafka-переходы без отделения от реализованного. Сейчас нет внешнего Gateway, а Storage — локальный blob-сервис; наличие protobuf не доказывает готовность end-to-end потока.

**Исправить:** обозначить текущий gRPC-стенд и целевую архитектуру отдельно. Поддержку S3 описывать списком реально проверенных операций/клиентов после реализации Gateway, а проектные схемы подписать как проектные.

### D03 · P2 · Неверные пути, порты и состав запуска

**Статус: ✅ исправлено в PR.**

**Где:** [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md), [GETTING_STARTED.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/GETTING_STARTED.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md), [services/users/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/users/README.md).

Users использует порт 50054, AuthZ — 50051: указание Users 50051 направляет клиента в другой сервис. Старые `proto/`, `shared/pkg/kit`, отдельная синхронизация AuthZ-прото не соответствуют текущему `shared/api`, `shared/pkg/go`, `shared/pkg/py` и отдельным kit-пакетам. В onboarding отсутствуют Metadata, Quota и migrator-metadata в перечне запускаемых компонентов.

**Исправить:** переписать таблицы и команды по текущим Compose/Makefile, явно различить адрес внутри Docker и адрес на хосте. Не копировать примеры credentials из старого service README поверх действующего локального конфига.

### D04 · P3 · Машинные ссылки и устаревшие исторические документы

**Статус: ✅ исправлено в PR.**

**Где:** [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md), удалённый `docs/auth-users-integration.md`, [docs/wiki-audit-2026-04-02.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/wiki-audit-2026-04-02.md), [docs/users-service-unit-test-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/users-service-unit-test-plan.md).

Абсолютная ссылка на локальный `GETTING_STARTED.md` не открывается у читателя GitHub. Апрельские документы содержат утверждения своего времени: Metadata отсутствует, Storage placeholder, старые пути kit/Neo4j, отсутствие части AuthZ-валидаций. План Users-тестов уже во многом реализован.

**Исправить:** локальную ссылку заменить репозиторной; лишний integration snapshot удалить по решению владельца. У оставшихся планов сохранить исходную дату, добавить дату актуализации и короткий срез текущего состояния, не переписывая прошлое задним числом.

### D05 · P1 · Users приписана отсутствующая защита RPC

**Статус: ✅ исправлено в PR.**

**Где:** [services/users/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/users/README.md), [docs/class_diagram.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/class_diagram.md); связанные обзорные схемы.

README обещает TLS, JWT и доступ только к собственному профилю. В [services/users/internal/app/app.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/users/internal/app/app.go) нет соответствующих транспортных credentials/JWT-проверок; handlers принимают `user_id`. Получивший сетевой доступ клиент не становится автоматически ограниченным своим профилем.

**Исправить:** описать Users как внутренний RPC-сервис и существующую границу доверия. TLS/JWT/self-only — отдельные требования к будущей защите, а не реализованные гарантии.

### D06 · P2 · Users API и события описаны по старой реализации

**Статус: ✅ исправлено в PR.**

**Где:** [services/users/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/users/README.md), [docs/class_diagram.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/class_diagram.md), [docs/object_diagram.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/object_diagram.md), [docs/sequence_diagrams.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/sequence_diagrams.md).

Заявлены `user.created`/`user.deleted`, старые методы и зависимости; текущий service использует repository/transaction manager без этих Kafka-публикаций. `Update` и `UpdatePassword` — разные RPC; Create не выдаёт JWT. Примеры health/адресов также направлены на старую конфигурацию.

**Исправить:** синхронизировать контракт с [shared/api/user/v1/user.proto](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/shared/api/user/v1/user.proto) и [services/users/internal/service/user/service.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/users/internal/service/user/service.go). События оставить как будущий сценарий, если они нужны.

### D07 · P2 · Auth README предлагает несуществующий локальный HTTP-вход

**Статус: ✅ исправлено в PR.**

**Где:** [services/auth/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/auth/README.md).

HTTP curl через Envoy/8080, старые пути сертификатов/прото и `user-server:50051` не соответствуют текущему Compose. Запуск имеющегося стенда не создаёт описанные HTTP endpoints.

**Исправить:** основной quick start строить на реальном gRPC и текущем Users endpoint; HTTP/Envoy пример убрать из рабочего сценария либо пометить как отдельный проектируемый deployment с необходимыми файлами.

### D08 · P1 · Login, refresh rotation и JWT claims описаны неверно

**Статус: ✅ исправлено в PR.**

**Где:** [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md), [services/auth/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/auth/README.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md), [docs/sequence_diagrams.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/sequence_diagrams.md); Wiki Gateway и sequence diagrams.

Login выдаёт refresh token; access получают отдельным RPC. Выпуск нового refresh не отзывает предыдущий. Claims используют строковый UUID `user_id` и `token_type`; чтение `sub` или числового ID из приведённых схем не соответствует текущему контракту.

**Исправить:** описать фактический обмен токенов и отсутствие revocation; rotation с инвалидированием — будущая #66. Основание: [services/auth/internal/service/auth/login.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/auth/internal/service/auth/login.go), [services/auth/internal/service/auth/get_refresh_token.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/auth/internal/service/auth/get_refresh_token.go), [services/auth/internal/model/model.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/auth/internal/model/model.go).

### D09 · P2 · Redis sessions и защита от DDoS преувеличены

**Статус: ✅ исправлено в PR.**

**Где:** [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md), [services/auth/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/auth/README.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md).

Redis Auth используется для учёта попыток входа, а не реестра активных JWT-сессий. Текущий rate limiter локален процессу: несколько реплик не разделяют единый лимит. Такой middleware сам по себе не является общей защитой от DDoS.

**Исправить:** назвать точный механизм и область действия; распределённый лимит и реестр токенов отделить как план.

### D10 · P1 · AuthZ: существование PARENT_OF ошибочно принимается за наследование прав

**Статус: ✅ исправлено в PR.**

**Где:** [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md), [services/authz/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/CLAUDE.md); Wiki Home/AuthZ/Gateway/Sequence Diagrams.

Хранение `PARENT_OF` не означает, что `CheckPermission` проходит по нему. Текущий запрос использует `MEMBER_OF` и разрешение на конечный ресурс; автоматически переносить bucket grant на дочерний object нельзя. Описания friend-based доступа также не подкреплены публичным контрактом.

**Исправить:** отразить действующий обход; наследование bucket→object описать отдельным решением с тестами. Основание: [services/authz/internal/repositories/neo4j/store.py](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/internal/repositories/neo4j/store.py).

### D11 · P1 · OWNER_OF и уровни прав расходятся с публичным API

**Статус: ✅ исправлено в PR.**

**Где:** [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md), [services/authz/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/CLAUDE.md); Wiki AuthZ/Gateway и диаграммы выдачи прав.

`OWNER_OF` поддерживается как legacy-ребро в store, но отсутствует среди разрешённых relation в gRPC. Публичный способ дать admin — `HAS_PERMISSION` с соответствующим уровнем. Формулировка, будто write даёт create/delete/admin, переворачивает иерархию: write включает read, но не более высокие уровни.

**Исправить:** примеры переписать на enum-контракт [shared/api/authz/v1/authz.proto](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/shared/api/authz/v1/authz.proto) и отдельно перечислить legacy-поддержку.

### D12 · P1 · Нет единой модели ресурса и действия для S3-авторизации

**Статус: 📌 подготовлен единый проект mapping; принятие policy и реализация остаются открытыми.**

**Где:** [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md); Wiki AuthZ/Gateway/Sequence Diagrams; #51–#58.

`object:{bucket}` не равнозначен `bucket:{bucket}`. Для нового ключа объект ещё отсутствует, для overwrite уже существует; слепая одна проверка не определяет обе политики. Маппинг create/write/admin для bucket и multipart расходится между текстами и комментариями proto.

**Исправить:** согласовать одну таблицу «S3 operation → resource → required action», включая новый объект, overwrite, initiation/abort и удаление bucket. Это решение о политике доступа, а не механическая замена слова.

### D13 · P2 · AuthZ: несовпадающие RPC-примеры, валидация и коды ошибок

**Статус: ✅ исправлено в PR.**

**Где:** [services/authz/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/README.md), [services/authz/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/CLAUDE.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md); Wiki AuthZ.

В старых примерах встречается `rebac.authz.v1` вместо `opens3.authz.v1`, строковые контракты вместо protobuf enums и обещание проверки префиксов entity IDs. Реализована валидация action/relation/level, но не всех этих IDs; исключения зависимостей не везде переводятся в обещанные коды. Delete отсутствующего tuple возвращает `success=false`, а не обязательно `NOT_FOUND`.

**Исправить:** отделить реально обеспеченные ошибки от желаемых; привести команды к актуальному proto и servicer.

### D14 · P2 · AuthZ audit topic, payload и env не соответствуют коду

**Статус: ✅ исправлено в PR.**

**Где:** [services/authz/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/README.md), [services/authz/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/CLAUDE.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md); Wiki AuthZ/Kafka.

Создаётся `AuditProducer` с default topic `auth-changes`; решения и tuple events отправляются туда же. Обещанный отдельный `auth-audit` и поля `timestamp_ms`, `from_cache`, `level` не совпадают с фактическими payload. Указанные `CACHE_TTL_SECONDS`, `KAFKA_AUDIT_TOPIC`, `CHANGES_TOPIC` не управляют текущим кодом; TTL задан числом 30.

**Исправить:** документировать реальные настройки/события, желаемое разделение вынести в задачу. Основание: [services/authz/internal/container.py](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/internal/container.py), [services/authz/internal/config.py](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/internal/config.py), [services/authz/internal/repositories/kafka/producer.py](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/internal/repositories/kafka/producer.py), [services/authz/internal/permission/service.py](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/internal/permission/service.py).

### D15 · P1 · Немедленный отзыв доступа через Kafka не обеспечен

**Статус: ✅ исправлено в PR.**

**Где:** [services/authz/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/README.md); Wiki AuthZ и Cache Strategy.

Основной Docker entrypoint запускает gRPC, а invalidator не включён отдельным сервисом в Compose. Даже при его запуске group membership changes не инвалидируют все зависимые resource decisions; гонка между проверкой и заполнением кеша также остаётся. Producer не делает изменение графа и событие одной транзакцией.

**Исправить:** убрать обещание немедленной инвалидации, описать TTL/best-effort и явный запуск consumer. Для строгого revoke требуется отдельный протокол. Технические основания — C05.

### D16 · P2 · AuthZ quick start содержит невыполнимые шаги

**Статус: ✅ исправлено в PR.**

**Где:** [services/authz/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/README.md), [services/authz/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/CLAUDE.md).

`cd rebac-auth-service`, старый `deploy/local/docker-compose`, сборка Docker из каталога сервиса не соответствуют монорепозиторию и root build context. Запуск всех сервисов, затем локального AuthZ на том же порту создаёт конфликт. Требуемость пароля описана иначе, чем default в конфиге.

**Исправить:** дать два самостоятельных сценария: всё в Docker либо зависимости в Docker + сервис локально. Использовать актуальные пути и не включать раскрытие секретов в диагностические команды.

### D17 · P3 · AuthZ: ограничения Python выданы за невозможность

**Статус: ✅ исправлено в PR.**

**Где:** [services/authz/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/README.md), [services/authz/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/CLAUDE.md).

Утверждение о невозможности consumer и gRPC в одном процессе неверно: это вопрос потоков/event loop и жизненного цикла. Каталог `internal` в Python также не обеспечивает запрет импорта, подобный Go.

**Исправить:** назвать разделение процессов архитектурным выбором, а `internal` — принятой структурой проекта.

### D18 · P1 · Quota CheckQuota уже резервирует расход: пример начисляет его дважды

**Статус: ✅ исправлено в PR.**

**Где:** [services/quota/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/README.md), [services/quota/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/CLAUDE.md); Wiki Quota и Gateway.

После успешного `CheckQuota(delta)` вызов `UpdateUsage(+delta)` снова увеличивает расход. В ручном сценарии это превращает 50 MiB в 100 MiB, а один объект/bucket — в два; дальнейшие ожидаемые ответы неверны.

**Исправить:** явно назвать CheckQuota «check-and-reserve»: при успехе операции повторно не начислять ту же дельту, при неуспехе компенсировать резерв. Не выдавать этот простой протокол за защищённый от сетевых retries: для этого нужны operation/reservation ID и reconciliation. Основание: [services/quota/src/service/quota.rs](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/src/service/quota.rs), [services/quota/src/cache/memory.rs](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/src/cache/memory.rs).

### D19 · P1 · Атомарность Quota чрезмерно обобщена

**Статус: ✅ исправлено в PR.**

**Где:** [services/quota/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/README.md), [services/quota/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/CLAUDE.md); Wiki Quota.

Проверка с резервом атомарна для записи одного subject в памяти одного процесса. User и bucket обновляются отдельными шагами, limits живут отдельно, несколько Quota replicas имеют независимую память. Это не распределённая транзакция и не единый строгий лимит кластера.

**Исправить:** указать область атомарности, single-writer/одна реплика как ограничение текущей модели, rollback и неизвестный исход RPC. Масштабирование требует согласованной архитектуры учёта.

### D20 · P2 · Quota: DashMap и показатели производительности описаны недостоверно

**Статус: ✅ исправлено в PR.**

**Где:** [services/quota/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/README.md), [services/quota/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/CLAUDE.md); Wiki Quota.

`DashMap::new()` не фиксирует 256 шардов; чтения используют блокировки. «CheckQuota ~200 ns» нельзя переносить с операции в памяти на gRPC-вызов. Приведённые latency, cold-start, memory/binary-size числа не сопровождаются воспроизводимым замером.

**Исправить:** убрать lock-free/фиксированные 256, назвать оценку гипотезой и отделить microbenchmark от RPC/нагрузочного теста. Основание: [services/quota/src/cache/memory.rs](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/src/cache/memory.rs) и [API DashMap](https://docs.rs/dashmap/latest/dashmap/struct.DashMap.html).

### D21 · P1 · Quota durability не ограничена гарантированно одной секундой

**Статус: ✅ исправлено в PR.**

**Где:** [services/quota/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/README.md), [services/quota/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/CLAUDE.md); Wiki Quota.

Интервал flush — не верхняя граница потери. После сбоя flush dirty-снимок может быть потерян (C04); Redis в dev работает без persistence. Даже будущий AOF everysec не устраняет предшествующее окно памяти сервиса и сетевые отказы. В Wiki AOF описан как действующая гарантия, тогда как README отдельно признаёт его отсутствие.

**Исправить:** описать фактический период по config, режим Redis и отсутствие строгой durability; не обещать «максимум 1 s». AOF — только часть будущего решения.

### D22 · P2 · Quota lifecycle и локальная сборка описаны неполно

**Статус: ✅ исправлено в PR.**

**Где:** [services/quota/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/README.md), [services/quota/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/CLAUDE.md); Wiki Quota.

`DeleteSubject` уже есть в proto/handler, а CLAUDE называет его не подключённым dead code. Удаление dirty-флага не гарантирует отсутствие последующей записи ранее снятого flush-снимка. Для локальной сборки нужен protoc через build.rs, его нет в полном перечне prerequisites.

**Исправить:** актуализировать API/сборку и не обещать защиту от resurrection без координации удаления с in-flight flush. Основание: [shared/api/quota/v1/quota.proto](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/shared/api/quota/v1/quota.proto), [services/quota/src/transport/grpc.rs](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/src/transport/grpc.rs), [services/quota/build.rs](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/build.rs).

### D23 · P2 · Storage: несуществующая команда и неверное объяснение go.work

**Статус: ✅ исправлено в PR.**

**Где:** [services/storage/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/README.md).

`make generate-storage` отсутствует: есть отдельные Go/Python targets и общий `make generate`. Shared module указан в go.mod, а не существует только благодаря workspace; поэтому объяснение зависимости неточно. Обещание, что один push непременно исправит `go mod tidy`, требует доступной версии модуля и не следует из go.work.

**Исправить:** привести команды к [Makefile](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/Makefile), объяснить отдельно module dependency и локальное workspace resolution по [services/storage/go.mod](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/go.mod) / [go.work](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/go.work).

### D24 · P2 · План Storage противоречит текущему контракту и самому себе

**Статус: ✅ исправлено в PR.**

**Где:** [docs/storage-service-implementation-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-service-implementation-plan.md).

Выполненные этапы соседствуют с «сейчас заглушка»; blob ID приписан repository, хотя его выбирает service; `size > 0` запрещает поддерживаемый пустой blob; старые rangeStart/rangeEnd не совпадают с offset/length. Уже реализованное шардирование остаётся будущей задачей. Указания tags/Docker не относятся ко всем реальным FS/component tests.

**Исправить:** оформить исторический план с итоговым статусом или обновить в актуальную спецификацию; не смешивать два времени в одном разделе.

### D25 · P2 · Storage: неверный момент появления файла и удаления multipart session

**Статус: ✅ исправлено в PR.**

**Где:** [services/storage/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/README.md), [docs/storage-fs-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-fs-architecture.md), [docs/storage-service-implementation-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-service-implementation-plan.md); Wiki Data Node/pending-finalize.

Final part появляется после rename, а не с первого чанка. Обычный blob ID выбирается до записи; multipart blob ID связан с upload ID, а не независимо создаётся только при Complete. Текущий Storage cleanup происходит внутри Complete, без ожидания внешнего Metadata.Finalize.

**Исправить:** описать реальную последовательность staging → rename → completion marker → cleanup; вариант ожидания Metadata обозначить изменением протокола, связанным с #36.

### D26 · P1 · Atomic rename ошибочно превращён в полную crash durability

**Статус: ✅ исправлено в PR.**

**Где:** [docs/storage-fs-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-fs-architecture.md), [docs/storage-service-implementation-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-service-implementation-plan.md); Wiki Data Node и архитектурные варианты.

Код делает fsync файла и rename, но не fsync каталога. Кроме того, rename между разными файловыми системами не обеспечивает такой commit; конфиг допускает раздельные пути. Атомарная видимость имени не равна гарантии пережить потерю питания.

**Исправить:** явно ограничить обещание текущей реализации; durability требует отдельного порядка синхронизации и общей filesystem для rename. Основание: [services/storage/internal/repository/storage/write_helpers.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/internal/repository/storage/write_helpers.go), [services/storage/internal/repository/storage/store_blob.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/internal/repository/storage/store_blob.go), [Linux fsync](https://man7.org/linux/man-pages/man2/fsync.2.html).

### D27 · P2 · Storage test inventory устарел

**Статус: ✅ исправлено в PR.**

**Где:** [services/storage/tests.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/tests.md).

Имен `TestUploadPart_ReturnsInvalidArgumentOnUploadIDMismatch` и `TestUploadPart_ReturnsInvalidArgumentOnPartNumberMismatch` в текущих тестах нет. Они относятся к прежнему контракту, где metadata могла повторяться в чанках; теперь используется header/chunk.

**Исправить:** заменить эти пункты реальными проверками текущего stream-контракта. Остальные перечисленные имена найдены статической сверкой; это не означает, что все тесты запускались и прошли.

### D28 · P2 · Разные health API ошибочно представлены одной проверкой готовности

**Статус: ✅ исправлено в PR.**

**Где:** [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md), [services/storage/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/README.md), [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md), [docs/kubernetes-deployment-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/kubernetes-deployment-plan.md); Wiki Services.

Custom Storage HealthCheck проверяет filesystem; стандартный gRPC Health зарегистрирован со статическим SERVING и не эквивалентен ему. Аналогично нельзя приписывать стандартной проверке автоматическую диагностику Neo4j/PostgreSQL. У Auth/Users также нельзя вызывать выдуманный service-level RPC вместо имеющегося health service.

**Исправить:** привести точные RPC и смысл liveness/readiness; не заявлять dependency readiness по одному статическому SERVING. Unary interceptor не следует описывать как покрывающий все streaming RPC.

### D29 · P2 · Kafka-контракт Storage не выбран и часть cleanup-переходов пропущена

**Статус: ✅ исправлено в PR.**

**Где:** [docs/storage-service-implementation-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-service-implementation-plan.md), [docs/storage-fs-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-fs-architecture.md); Wiki Kafka/архитектурные варианты.

Смешаны `object-stored`/`object-deleted` и `object-blob-stored`/`object-delete-requested`. `object-aborted` нужен в описанном abort flow, но отсутствует в обязательном списке. `kafka:9092` как внутренний bootstrap расходится с внутренним listener 29092.

**Исправить:** одна таблица topic → producer → consumer → schema → retry/idempotency; старые имена обозначить историческими. Отдельно указать, что Storage Kafka-интеграция ещё проектируется.

### D30 · P2 · Deployment CI приведён как рабочий, хотя это неполный эскиз

**Статус: ✅ исправлено в PR.**

**Где:** [docs/kubernetes-deployment-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/kubernetes-deployment-plan.md).

В YAML jobs отсутствует обязательный `runs-on`; для deploy не показаны checkout, доступ к registry и Kubernetes credentials. ServiceMonitor рассчитывает на прямой `/metrics`, тогда как существующая наблюдаемость использует OTLP collector.

**Исправить:** либо дополнить до проверяемого примера под текущий стек, либо назвать псевдоконфигурацией с перечнем недостающих компонентов. Не обещать, что copy/paste разворачивает проект.

### D31 · P2 · Kubernetes StatefulSet и размещение не обеспечивают заявленную HA

**Статус: ✅ исправлено в PR.**

**Где:** [docs/kubernetes-deployment-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/kubernetes-deployment-plan.md).

В Storage StatefulSet нет необходимого selector и соответствующих template labels. Не заданы правила размещения реплик по узлам. На схеме по два Kafka/Neo4j оказываются на одном узле, PG-meta primary/replica — на одном; падение узла может уничтожить большинство/обе копии. Единственный показанный Sentinel не задаёт отказоустойчивый quorum.

**Исправить:** labels/selector, anti-affinity или topology spread, согласованные кворумы и failure domains; пересчитать число pod из итоговой таблицы, а не оставлять `36 = 20 + 16`. Основание: [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/).

### D32 · P2 · Chaos-примеры и инфраструктурные оценки чрезмерно уверенные

**Статус: ✅ исправлено в PR.**

**Где:** [docs/kubernetes-deployment-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/kubernetes-deployment-plan.md).

`kubectl drain` — контролируемое выселение, а не crash узла; `--force` не означает игнорировать PDB. PDB не защищает от всех сбоев/прямых удалений. Команда iptables требует прав/утилиты и не доказывает изоляцию всех Gateway pods. Bitnami postgresql-ha использует repmgr/pgpool, а не Patroni. Цены без региона/даты/состава расходов и категорическое сравнение облаков не обоснованы.

**Исправить:** различить eviction/crash/network partition; обозначить исполнимые prerequisites, точный chart и историческую оценку расходов вместо обещания цены. Основания: [Pod disruptions](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/), [postgresql-ha Chart](https://github.com/bitnami/charts/blob/main/bitnami/postgresql-ha/Chart.yaml).

## Wiki и общие архитектурные утверждения

В этой части названия страниц кликабельны и ведут на опубликованную Wiki. Для совпадающих утверждений в `docs/` исправление должно быть синхронным.

### W01 · P1 · W + R > N не является достаточным доказательством strong consistency

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [docs/distributed-storage-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/distributed-storage-architecture.md); [Architecture](https://github.com/alesplll/opens3-rebac/wiki/Architecture), [Основная-проблема-доставки-и-варианты-ее-решения](https://github.com/alesplll/opens3-rebac/wiki/%D0%9E%D1%81%D0%BD%D0%BE%D0%B2%D0%BD%D0%B0%D1%8F-%D0%BF%D1%80%D0%BE%D0%B1%D0%BB%D0%B5%D0%BC%D0%B0-%D0%B4%D0%BE%D1%81%D1%82%D0%B0%D0%B2%D0%BA%D0%B8-%D0%B8-%D0%B2%D0%B0%D1%80%D0%B8%D0%B0%D0%BD%D1%82%D1%8B-%D0%B5%D0%B5-%D1%80%D0%B5%D1%88%D0%B5%D0%BD%D0%B8%D1%8F).

Формула гарантирует пересечение кворумов при соответствующих предпосылках, но сама по себе не решает concurrent writes, порядок версий, выбор ответа и stale topology. При N=3/W=2 чтение с произвольной одной реплики может попасть в не записанную копию.

**Исправить:** для immutable blobs явно задать чтение с подтверждённых держателей/retry и согласованный Metadata pointer; гарантии вывести из протокола, а не из одной формулы. Проектный дизайн не называть устройством AWS S3.

### W02 · P1 · io.MultiWriter не даёт независимую запись до fastest W

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [docs/distributed-storage-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/distributed-storage-architecture.md).

`io.MultiWriter(pipeWriters...)` последовательно вызывает каждый writer. Заблокировавшаяся pipe одной реплики тормозит входной stream; схема не обеспечивает обещанную задержку по W самым быстрым узлам автоматически.

**Исправить:** описать bounded fan-out, backpressure, отмену медленных реплик и достижение полного W-commit; убрать неверную гарантию у данного фрагмента. Основание: [реализация io.MultiWriter](https://go.dev/src/io/multi.go).

### W03 · P1 · Для реплик нельзя оставить Storage API неизменным и ожидать одинаковый blob_id

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [docs/distributed-storage-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/distributed-storage-architecture.md); [Architecture](https://github.com/alesplll/opens3-rebac/wiki/Architecture), [Services-—-Data-Node](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Data-Node).

Каждый StoreObject самостоятельно создаёт UUID; request не позволяет задать общий ID. Несколько независимых вызовов на разные узлы не создают копии с одним логическим blob_id.

**Исправить:** выбрать принимаемый Storage идентификатор/operation ID с защитой от конфликтов либо хранить mapping логического blob к отдельным node-local IDs. Это изменение контракта, а не «Storage остаётся как есть». Основание: [shared/api/storage/v1/storage.proto](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/shared/api/storage/v1/storage.proto).

### W04 · P1 · Статус suspect — ещё не fencing

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [docs/distributed-storage-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/distributed-storage-architecture.md).

Один новый Gateway перестанет выбирать suspect node, но старый coordinator может продолжить запись по старой топологии. Проверка checksum после возврата узла не предотвращает такие записи.

**Исправить:** называть suspect исключением из размещения. Для fencing определить монотонный epoch/token и проверку на стороне исполнителя, правила смены authority и reject устаревшей операции.

### W05 · P2 · Stateless placement и аналоги GFS/Ceph описаны слишком свободно

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [docs/distributed-storage-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/distributed-storage-architecture.md).

Data-path proxy может не хранить долговечное состояние локально, но topology, heartbeats, repair/migration требуют согласованного состояния и владельца решений. GFS/Ceph нельзя приводить как буквальный пример центрального placement-proxy для потока данных. В цепочке возможен pipeline, поэтому простое умножение полной latency на длину цепи не является универсальной оценкой.

**Исправить:** отделить stateless frontend от stateful control plane, назвать аналоги ограниченными и заменить категорические задержки моделью с явными предпосылками. Основания для сравнения: [GFS, разделы 2.1 и 3.2](https://static.googleusercontent.com/media/research.google.com/en/us/archive/gfs-sosp2003.pdf), [Ceph architecture](https://docs.ceph.com/en/quincy/architecture/).

### W06 · P1 · DB commit + Kafka publish не гарантирует отсутствие потерянных событий

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-—-Utils-—-Kafka](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Utils-%E2%80%94-Kafka), [Architecture](https://github.com/alesplll/opens3-rebac/wiki/Architecture), [Architecture-‐-Sequence-Diagrams](https://github.com/alesplll/opens3-rebac/wiki/Architecture-%E2%80%90-Sequence-Diagrams); [docs/sequence_diagrams.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/sequence_diagrams.md).

Kafka retries/ack не закрывают окно между изменением БД и отправкой события. В Metadata это текущий разрыв (C02); consumer cleanup Storage/AuthZ ещё не образует готовый end-to-end cascade.

**Исправить:** описать фактический best-effort и будущий transactional outbox, at-least-once delivery и идемпотентных consumers. Не обещать end-to-end exactly-once на основании использования Kafka.

### W07 · P1 · Событие удаления не содержит данных, на которые рассчитывает AuthZ cleanup

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-—-Utils-—-Kafka](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Utils-%E2%80%94-Kafka), [Services-—-Metadata](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Metadata), [Services-—-AuthZ](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-AuthZ).

Фактическое object event — `object_id, blob_id`; bucket event — `bucket_id, bucket_name`. Текст ожидает также key/owner/timestamps/type. После удаления Metadata нельзя полагаться на запрос удалённой строки, чтобы получить `bucket/key` для AuthZ.

**Исправить:** согласовать immutable payload для каждой цели очистки либо перейти на стабильные resource IDs. Показать текущую и проектную схемы отдельно. Основание: [services/metadata/internal/service/object/delete.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/internal/service/object/delete.go), [services/metadata/internal/service/bucket/delete.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/internal/service/bucket/delete.go).

### W08 · P1 · Поздний delete по bucket/key может стереть права нового объекта

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-—-Utils-—-Kafka](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Utils-%E2%80%94-Kafka), [Architecture-‐-Sequence-Diagrams](https://github.com/alesplll/opens3-rebac/wiki/Architecture-%E2%80%90-Sequence-Diagrams); #55/#56.

Удалили ресурс, создали новый с тем же именем, затем пришёл старый cleanup event. Очистка по имени удалит новые связи. Идемпотентность повторения старого события не решает это смешение поколений.

**Исправить:** generation/resource UUID, проверка принадлежности удаляемого состояния конкретному поколению и правила порядка событий; не применять wildcard cleanup по переиспользуемому имени без такой защиты.

### W09 · P1 · Delete marker ошибочно ведёт к физическому удалению старых данных

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-—-Metadata](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Metadata), [Services-—-Data-Node](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Data-Node), [Services-—-Utils-—-Kafka](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Utils-%E2%80%94-Kafka); [docs/storage-service-implementation-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-service-implementation-plan.md).

В versioning-enabled bucket обычный DELETE добавляет delete marker, а предыдущие версии остаются. Схемы, удаляющие их blob как следствие marker, уничтожают историю. Marker сам не содержит blob; проектное `blob_id NOT NULL` с marker несовместимо.

**Исправить:** различить hide current, permanent delete конкретной версии и GC. Не начислять освобождение её bytes в Quota при одном marker. Основание: [Delete markers](https://docs.aws.amazon.com/AmazonS3/latest/userguide/DeleteMarker.html).

### W10 · P2 · Metadata Wiki не соответствует существующему proto и схеме

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-—-Metadata](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Metadata), [Services](https://github.com/alesplll/opens3-rebac/wiki/Services), [Architecture](https://github.com/alesplll/opens3-rebac/wiki/Architecture); [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md).

GetBucket возвращает вложенный `bucket`, а не показанный плоский объект; есть HeadBucket. Схема уже содержит lifecycle/версии, старые `is_deleted`-описания не передают текущую модель. «ListBuckets всегда OK» игнорирует ошибки зависимостей; bucket-level включение versioning пока не реализовано. В таблице зависимостей Metadata нельзя оставлять прочерк: нужны PostgreSQL, миграции и Kafka.

**Исправить:** актуализировать по [shared/api/metadata/v1/metadata.proto](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/shared/api/metadata/v1/metadata.proto) и [services/metadata/migrations/20260423000000_init.sql](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/migrations/20260423000000_init.sql), явно отметить C01.

### W11 · P1 · В pending/finalize варианте потерян шаг durable promotion

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [[?]-Архитектура-"Metadata-authoritative---pending-finalize"](https://github.com/alesplll/opens3-rebac/wiki/%5B%3F%5D-%D0%90%D1%80%D1%85%D0%B8%D1%82%D0%B5%D0%BA%D1%82%D1%83%D1%80%D0%B0-%22Metadata-authoritative---pending-finalize%22), [[?]-Архитектура-"Event-driven-commit-and-outbox-inbox"](https://github.com/alesplll/opens3-rebac/wiki/%5B%3F%5D-%D0%90%D1%80%D1%85%D0%B8%D1%82%D0%B5%D0%BA%D1%82%D1%83%D1%80%D0%B0-%22Event-driven-commit-and-outbox-inbox%22), [Основная-проблема-доставки-и-варианты-ее-решения](https://github.com/alesplll/opens3-rebac/wiki/%D0%9E%D1%81%D0%BD%D0%BE%D0%B2%D0%BD%D0%B0%D1%8F-%D0%BF%D1%80%D0%BE%D0%B1%D0%BB%D0%B5%D0%BC%D0%B0-%D0%B4%D0%BE%D1%81%D1%82%D0%B0%D0%B2%D0%BA%D0%B8-%D0%B8-%D0%B2%D0%B0%D1%80%D0%B8%D0%B0%D0%BD%D1%82%D1%8B-%D0%B5%D0%B5-%D1%80%D0%B5%D1%88%D0%B5%D0%BD%D0%B8%D1%8F); [docs/storage-fs-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-fs-architecture.md).

Если данные названы staging, а затем Metadata делает объект видимым, нужно определить, кто и когда делает их доступными по конечному blob ID. Также одного Storage event недостаточно восстановить bucket/key/version после сбоя, если намерение и корреляция нигде долговечно не сохранены.

**Исправить:** добавить durable intent, точный идентификатор версии/операции, storage commit acknowledgment, finalize и только затем окончательный успех. Если StoreObject уже выполняет promotion, назвать его так, не оставлять второй несуществующий этап.

### W12 · P2 · «Пустой bucket» не означает отсутствие всех файлов на диске

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-—-Metadata](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Metadata), [Services-—-Data-Node](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Data-Node), [Architecture-‐-Sequence-Diagrams](https://github.com/alesplll/opens3-rebac/wiki/Architecture-%E2%80%90-Sequence-Diagrams).

Каталог может быть пуст при ещё не выполненном async GC, orphan blobs и multipart staging. Проверка только текущих объектов также не учитывает сохранённые версии.

**Исправить:** определить логическую пустоту и отдельный physical GC; для versioned bucket учитывать версии/delete markers. Не строить correctness удаления bucket на предположении, что Storage организован по его имени.

### W13 · P1 · Оптимизации AuthZ cache могут продлевать уже отозванный доступ

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [[?]-AuthZ-‐-Cache-Strategy](https://github.com/alesplll/opens3-rebac/wiki/%5B%3F%5D-AuthZ-%E2%80%90-Cache-Strategy).

Рекомендация длинного/sliding ALLOW как безопасного опирается на гарантированную инвалидацию, которой нет (D15/C05). Итоговый пример продлевает TTL без реализации описанного hard deadline. Fallback к bucket/admin cache также меняет политику наследования относительно текущего store.

**Исправить:** не рекомендовать эту схему к внедрению до выбора revoke SLA, hard bound и dependency invalidation. Семантику наследования вынести из раздела «оптимизация» в контракт авторизации; мониторинг lag сам по себе не делает revoke мгновенным.

### W14 · P2 · Ошибки в технических примерах cache strategy

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [[?]-AuthZ-‐-Cache-Strategy](https://github.com/alesplll/opens3-rebac/wiki/%5B%3F%5D-AuthZ-%E2%80%90-Cache-Strategy).

Обычный Redis pipeline не равнозначен атомарной транзакции; нужен MULTI/EXEC либо Lua. `EXAT` появился в Redis 6.2, а не 7. `if cached := ...` не обрабатывает кешированный False как hit. Предлагаемый L1 нуждается в синхронизации при доступе из потоков gRPC.

**Исправить:** точные примитивы, проверка `is not None`, thread-safe доступ к L1, воспроизводимые benchmarks вместо обещанного hit rate/наносекунд. Основание: [Redis transactions/pipelines](https://redis.io/docs/latest/develop/clients/rust/transpipe/), [SET history](https://redis.io/docs/latest/commands/set/), [cachetools thread safety](https://cachetools.readthedocs.io/en/latest/).

### W15 · P2 · Лимиты проекта ошибочно названы текущими лимитами AWS S3

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-‐-Quota](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%90-Quota), [Services-—-Gateway](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Gateway); связанные ограничения в README.

100 buckets/account и максимум объекта 5 TB устарели как описание AWS: текущий default bucket quota — 10 000, multipart object limit — 48.8 TiB. Это не требует увеличивать учебный стенд до таких размеров.

**Исправить:** отделить собственные настроенные ограничения от актуальных внешних. 507/`QuotaExceeded` назвать расширением проекта, а не универсальным стандартным S3-ответом; убрать расхождение 400/507 внутри одной политики. Источники: [bucket quotas](https://docs.aws.amazon.com/AmazonS3/latest/userguide/BucketRestrictions.html), [multipart limits](https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html).

### W16 · P1 · На схемах изменения прав отсутствует авторизация инициатора

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Services-—-Gateway](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Gateway), [Architecture-‐-Sequence-Diagrams](https://github.com/alesplll/opens3-rebac/wiki/Architecture-%E2%80%90-Sequence-Diagrams); [docs/sequence_diagrams.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/sequence_diagrams.md).

Подпись актёра Owner не проверяет его право выполнить grant/revoke. Прямая отправка WriteTuple после одной аутентификации позволяет читателю реализовать выдачу прав без admin-check.

**Исправить:** явно поставить проверку полномочий на целевой ресурс перед изменением графа и определить, какой компонент обязан её выполнить. Внутренний AuthZ RPC не следует выдавать за безопасный публичный endpoint.

### W17 · P2 · Термины strong consistency, 2PC и eventual consistency смешаны

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Glossary](https://github.com/alesplll/opens3-rebac/wiki/Glossary), [Architecture](https://github.com/alesplll/opens3-rebac/wiki/Architecture), архитектурные варианты.

2PC решает атомарность commit участников, но сам по себе не обеспечивает изоляцию/линеаризуемость. Eventual convergence требует условий доставки/прекращения конфликтующих обновлений. Асинхронный внутренний шаг не означает автоматически слабую видимость для клиента. Request correlation ID и стабильный operation ID для retries имеют разные задачи.

**Исправить:** короткие определения с границами гарантии, применённые к выбранному протоколу проекта.

### W18 · P3 · Навигация и статус Roadmap требуют актуализации

**Статус: ✅ исправлено в опубликованной Wiki (`009dbd2..db3a1c7`).**

**Где:** [Home](https://github.com/alesplll/opens3-rebac/wiki/Home), [Roadmap](https://github.com/alesplll/opens3-rebac/wiki/Roadmap), [_Footer](https://github.com/alesplll/opens3-rebac/wiki/_Footer), ссылки на два архитектурных варианта и Cache Strategy.

Часть ссылок не учитывает фактический префикс `[?]-` и отличается от названий страниц; есть незавершённый `Next Page ?`. Roadmap и README по-разному отмечают выполненность одних этапов. Закрытие issue само по себе не доказательство работающего кода.

**Исправить:** сверить ссылки по реальным page names и каждому этапу дать статус на дату аудита. Либо сохранить исторический Roadmap с явной датой, либо сделать актуальный; не оставлять два конфликтующих «сейчас».

### W19 · P3 · Неподтверждённые свойства масштаба и сведения о команде

**Статус: ⚠️ техническое утверждение исправлено; состав команды оставлен владельцам на подтверждение.**

**Где:** [Home](https://github.com/alesplll/opens3-rebac/wiki/Home), [Team](https://github.com/alesplll/opens3-rebac/wiki/Team), [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md).

Flat namespace сам по себе не гарантирует работу с миллиардами объектов без деградации. Распределение участников/имён между Team и README различается; из кода нельзя надёжно установить, какое верно.

**Исправить:** масштаб обозначить целью, подкрепляемой нагрузочными тестами. Состав/роли команды — отдельное уточнение у владельца, не автоматическая «фактологическая» замена по предположению.

## Issues: исправления текста backlog

### I01 · P1 · #64: ETag не является идентификатором операции

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#64](https://github.com/alesplll/opens3-rebac/issues/64).

Два самостоятельных PUT одинаковых байтов в versioning-enabled bucket создают отдельные версии. ETag характеризует представление объекта и не является универсальным MD5 или retry token. Условие «одинаковый ETag → вернуть прежнюю версию» незаметно меняет API.

**Переформулировать:** «Внутренние повторяемые команды идентифицируются отдельным стабильным operation ID; повтор той же операции возвращает прежний результат. Независимая запись создаёт новую версию согласно режиму bucket». Источники: [PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html), [ETag](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingMetadata.html).

### I02 · P1 · #64: check-before-insert не защищает от concurrency

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#64](https://github.com/alesplll/opens3-rebac/issues/64).

Два запроса могут одновременно не найти существующий результат. Сам SELECT перед INSERT не делает operation idempotent. При этом blanket-описание текущего кода как вообще не использующего конкурентную защиту слишком широкое: есть upsert/транзакции.

**Переформулировать:** уникальное ограничение на scope + operation ID, обработка конфликта, fingerprint параметров, сохранённый terminal result; повтор ID с другими параметрами — ошибка. Для независимых overwrite определить порядок current pointer, не дедуплицировать их по bytes.

### I03 · P1 · #52/#54/#58/#65: окончательный успех нельзя отделить от заявленной видимости

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#52](https://github.com/alesplll/opens3-rebac/issues/52), [#54](https://github.com/alesplll/opens3-rebac/issues/54), [#58](https://github.com/alesplll/opens3-rebac/issues/58), [#65](https://github.com/alesplll/opens3-rebac/issues/65).

В #52/#58 ответ предшествует Metadata finalize; #54 предлагает дождаться видимости через Eventually; #65 меняет runtime, не фиксируя внешний success barrier. После завершённого успешного S3 object write последующий read/list должен видеть запись. Внутренний Kafka допустим, если конечный успех ждёт commit. Это не следует смешивать с отдельно eventual bucket configuration.

**Переформулировать:** минимальный стенд — durable blob → синхронная регистрация версии → финальный успешный ответ; альтернативный async lifecycle сохраняет тот же внешний барьер. Eventually оставить для GC/readiness. Источник: [S3 consistency](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html).

### I04 · P2 · #52/#65: проектный протокол назван устройством «реального S3»

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#52](https://github.com/alesplll/opens3-rebac/issues/52), [#65](https://github.com/alesplll/opens3-rebac/issues/65).

Документация AWS описывает внешний контракт, а не подтверждает именно предлагаемые два внутренних этапа. Текущий CreateObjectVersion требует уже записанный blob и создаёт committed version; смена его смысла на reservation затронет callers/proto. Pending intent до записи требует заранее согласованных IDs и отдельного хранения состояния.

**Переформулировать:** «Предлагаемый протокол OpenS3», явные API reserve/finalize/abort и план миграции, совместимость callers и I03. Не принимать наличие lifecycle-полей за готовность этого runtime.

### I05 · P2 · #52: Content-MD5 ещё не встроен в описанный Storage-контракт

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#52](https://github.com/alesplll/opens3-rebac/issues/52).

Нельзя просто передать ожидаемый digest в существующий StoreObject request: соответствующего поля нет. Проверка после окончательного успешного ответа уже не отклоняет повреждённый upload.

**Переформулировать:** выбрать проверку в Gateway при streaming или явное расширение Storage API; сравнить digest до окончательного commit/успеха, определить cleanup и `BadDigest`. Не приравнивать эту проверку ко всем возможным ETag.

### I06 · P1 · #53: pending overwrite не должен скрывать прежнюю committed версию

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#53](https://github.com/alesplll/opens3-rebac/issues/53).

Правило «pending object → 404» подходит новому ключу без committed version, но ломает чтение существующего объекта во время overwrite. Текущая схема не случайно различает current и pending pointers.

**Переформулировать:** GET выбирает committed current version; новая pending version не видна. 404 — когда читаемой current version нет либо текущая версия является delete marker. Смена указателя атомарна.

### I07 · P2 · #53: Flush и Range описаны слишком категорично

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#53](https://github.com/alesplll/opens3-rebac/issues/53).

Go не буферизует обязательно весь body до конца без Flush: буферы ограничены, промежуточный proxy также может буферизовать. Flush каждого чанка — настройка latency/overhead. Любой синтаксически неверный Range нельзя объяснять как обязательно 416 по RFC; нужно различать invalid и unsatisfiable, а также определить неподдерживаемый multi-range.

**Переформулировать:** bounded streaming, контролируемая политика flush, явная single-range parsing/error policy; 416 с полным размером для unsatisfiable диапазона. Источники: [net/http](https://pkg.go.dev/net/http), [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html), [S3 GetObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html).

### I08 · P1 · #51/#54: Bearer-only Gateway не совместим с неизменённым AWS SDK

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#51](https://github.com/alesplll/opens3-rebac/issues/51), [#54](https://github.com/alesplll/opens3-rebac/issues/54), историческая [#13](https://github.com/alesplll/opens3-rebac/issues/13).

Обычный S3 SDK использует SigV4 credentials/signature, а Bearer JWT — другой механизм. Один endpoint/path-style не делает эти механизмы взаимозаменяемыми.

**Переформулировать:** либо включить SigV4 в поддерживаемый контракт, либо явно назвать JWT-вход расширением проекта и показать custom client adapter. Тестирование модифицированного клиента не выдавать за штатную совместимость. Источник: [AWS authentication methods](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_sigv-authentication-methods.html).

### I09 · P2 · #54: тестовый план содержит неверные команды и слишком сильные выводы

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#54](https://github.com/alesplll/opens3-rebac/issues/54).

`--scale 0` не указывает сервис; `s3.WithPathStyle(true)` не соответствует конфигурации Go SDK v2. Полный smoke lifecycle требует #55/#56, а не только upload/download. Один Go SDK test не доказывает работу boto3/mc; фиксированная оценка ресурсов runner не универсальна.

**Переформулировать:** реальные SDK options `UsePathStyle`/`BaseEndpoint`, Compose `--scale service=0`, точные зависимости и отдельные compatibility cases. Read-after-write проверять сразу; E2E — свидетельство для покрытых случаев, а не доказательство всей совместимости. Источник: [SDK endpoints](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html), [S3 Options](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3#Options).

### I10 · P1 · #58: минимум part нельзя отклонять одинаково на каждом UploadPart

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#58](https://github.com/alesplll/opens3-rebac/issues/58).

Во время UploadPart ещё неизвестно, окажется ли эта часть последней в итоговом списке. Предварительно отклоняя любую часть <5 MiB, Gateway отклонит допустимую последнюю часть. Размеры частей в документации S3 — 5 MiB–5 GiB, исключение снизу для последней; GB/MB здесь двусмысленны.

**Переформулировать:** верхний предел и номер проверять при приёме, фактический размер учитывать при streaming; минимум для всех, кроме последней выбранной части, проверять на Complete. Источник: [multipart limits](https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html).

### I11 · P1 · #58: multipart ETag формула не универсальна

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#58](https://github.com/alesplll/opens3-rebac/issues/58).

MD5 от бинарных part-MD5 с суффиксом числа частей — распространённый вариант, но не универсальный контракт для всех режимов checksum/encryption. Утверждение, что любой SDK обязательно сочтёт иной ETag повреждением, неверно. Текущий full-blob MD5 Storage и внешний multipart ETag — разные поля/семантики.

**Переформулировать:** определить поддерживаемый режим, отдельные checksum/ETag и сохранение итогового ETag в Metadata; test vector формулы — только для выбранного режима. Источники: [multipart overview](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html), [CompleteMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CompleteMultipartUpload.html).

### I12 · P2 · #58: схема initiation и утверждение про SDK требуют исправления

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#58](https://github.com/alesplll/opens3-rebac/issues/58).

Диаграмма создаёт upload только в Metadata через пока отсутствующий CreateUpload, хотя Storage требует собственную session. Автоматический multipart относится к high-level transfer helpers, а не к любому вызову SDK PutObject. Непоследовательные номера допустимы не во всех checksum-режимах.

**Переформулировать:** Metadata tracking + Storage initiation, binding uploadId к bucket/key/инициатору, rollback частичного initiation, retry Complete/Abort. Указать конкретный поддерживаемый checksum mode и high-level helpers. Источник: [Boto3 transfer configuration](https://docs.aws.amazon.com/boto3/latest/guide/s3.html).

### I13 · P1 · #59: latest non-deleted version нарушает delete marker semantics

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#59](https://github.com/alesplll/opens3-rebac/issues/59).

Если current — delete marker, обычный GET не должен пропускать marker и возвращать более старые данные. Version ID непрозрачен клиенту: сортируемость не требование S3, UUID не запрещён. Внутренний порядок задаётся отдельно.

**Переформулировать:** GET current marker → 404; explicit GET marker version → 405; остальные версии остаются доступны по ID. Порядок хранить в catalog, не выводить из строки version ID. Источник: [Delete markers](https://docs.aws.amazon.com/AmazonS3/latest/userguide/DeleteMarker.html).

### I14 · P1 · #60: отзыв presigned URL и самодельная подпись описаны неверно

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#60](https://github.com/alesplll/opens3-rebac/issues/60).

AWS presigned URL не гарантированно действует после отзыва credentials/прав подписавшего. HMAC от собственного server secret с параметрами `X-Amz-*` не становится SigV4 только из-за названий параметров.

**Переформулировать:** реальный SigV4 presigning либо явно отдельный signed-URL протокол OpenS3. Подпись связывает method/resource/expiry и необходимые headers; задать модель revocation и проверки прав. Источник: [AWS presigned URL FAQ](https://docs.aws.amazon.com/prescriptive-guidance/latest/presigned-url-best-practices/faq.html).

### I15 · P2 · #57: показатели S3 ошибочно превращены в фиксированный rate limit

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#57](https://github.com/alesplll/opens3-rebac/issues/57).

3500 write/5500 read requests в секунду — опубликованный ориентир «не менее» на prefix, не общий жёсткий предел S3. S3 SlowDown использует 503; проектный 429 нельзя выдавать за тот же стандартный ответ. Недоступность Auth backend не доказывает неверность credentials.

**Переформулировать:** собственная политика лимитов и совместимый error mapping; invalid credentials отличать от unavailable backend. Если Quota fail-open допустим, добавить учёт последующей компенсации/reconciliation. Источник: [S3 performance](https://docs.aws.amazon.com/AmazonS3/latest/userguide/optimizing-performance.html).

### I16 · P1 · #55: bucket lifecycle не учитывает версии и concurrent writes

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#55](https://github.com/alesplll/opens3-rebac/issues/55).

Подсчёт active objects недостаточен для versioned bucket. ReadCommitted сам по себе не исключает запись между проверкой пустоты и удалением. Storage хранит blobs по ID, не по bucket. Возврат всех расшаренных buckets отличается от S3 ListBuckets, который перечисляет owned buckets.

**Переформулировать:** общая защита create-object/delete-bucket в catalog, версии/markers в условии пустоты, физический GC отдельно; shared-buckets listing как расширение. Дополнить bucket naming, где текущий список чрезмерно запрещает точки/не учитывает reserved names. Источник: [ListBuckets](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBuckets.html), [bucket naming rules](https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html).

### I17 · P1 · #56: cascade смешивает логическое и физическое удаление

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#56](https://github.com/alesplll/opens3-rebac/issues/56).

Delete marker не повод удалять старые blobs (W09). Soft delete без outbox не закрывает DB/event gap (C02). `object-stored` должен финализировать Metadata, а не восприниматься как Storage cleanup. Wildcard DeleteTuple отсутствует в текущем публичном API. Идемпотентность HTTP DELETE не означает, что versioned повтор обязан создавать ровно один marker.

**Переформулировать:** отдельные операции marker/permanent-version-delete, точные consumers/events и безопасная AuthZ cleanup с поколением ресурса. Видимость object DELETE и eventual physical GC описать раздельно.

### I18 · P1 · #49: retry по gRPC code не обеспечивает безопасное повторение

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#49](https://github.com/alesplll/opens3-rebac/issues/49).

DeadlineExceeded/Unavailable могут прийти после фактического commit. Слепой retry способен повторить резерв Quota, создать версию или выполнить иную мутацию дважды. Streaming дополнительно требует replayable body. Обёртка ошибки не обязательно сохраняет stack trace; пример Users→Auth не соответствует действующей зависимости Auth→Users.

**Переформулировать:** retry policy по методу, operation IDs/дедупликация где необходимы, bounded backoff/deadline и воспроизводимый stream; утверждения о внутренних hops AWS убрать как неподтверждённые.

### I19 · P1 · #36: удаление completion marker может сломать retry завершённой операции

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#36](https://github.com/alesplll/opens3-rebac/issues/36).

Если Metadata finalize прошёл, marker удалён, но финальный ответ клиент потерялся, повтор Complete должен найти сохранённый terminal result. Очистка marker без этого оставляет неопределённый результат. Текущий persisted BlobMeta не содержит обещанного `created_at`.

**Переформулировать:** хранение результата Complete в Metadata/ином durable registry на оговорённый retry horizon; GC marker только после обеспечения этой возможности. Возраст считать по действительно существующему источнику либо добавить поле явно.

### I20 · P2 · #61: исходное состояние observability описано как пустое

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#61](https://github.com/alesplll/opens3-rebac/issues/61).

В [infra/otel/grafana/dashboards](https://github.com/alesplll/opens3-rebac/tree/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/infra/otel/grafana/dashboards) уже есть dashboards, в сервисах — structured logging. План содержит несовпадающие пути/имена metrics. Один ошибочный запрос не гарантирует срабатывание правила sustained error-rate. Rebuild контейнера сам по себе не стирает сохранённый Grafana volume.

**Переформулировать:** gap analysis существующих dashboards/metrics, проверяемые alert scenarios с длительностью/нагрузкой, distinction provisioning и persisted manual state. RED не приписывать Google SRE: у Google другой набор golden signals. Основание: [объяснение Tom Wilkie](https://grafana.com/blog/what-is-observability-best-practices-key-metrics-methodologies-and-more/).

### I21 · P2 · #62: перечень существующих CI checks устарел

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#62](https://github.com/alesplll/opens3-rebac/issues/62).

Сервисных workflows шесть, а не пять. Уже есть часть Quota lint/Redis tests и AuthZ coverage/integration; корректная задача — заполнить оставшиеся пробелы, в частности Metadata integration. В scanner scope указаны HIGH+CRITICAL, а acceptance блокирует только CRITICAL.

**Переформулировать:** таблица «есть / добавить» по фактическим [.github/workflows](https://github.com/alesplll/opens3-rebac/tree/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/.github/workflows), согласованный threshold сканирования, команды и зависимости.

### I22 · P2 · #63: p99 назван worst case, а benchmark не отделён от модели хранения

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#63](https://github.com/alesplll/opens3-rebac/issues/63).

p99 — процентиль, не максимальная задержка; mean не бессмысленен, просто недостаточен. Kafka lag в offsets/messages не равен времени ожидания. tmpfs-результат не характеризует latency/durability дискового Storage. Предложенный workload mix не универсален для всех S3-систем.

**Переформулировать:** percentile/max/mean отдельно, lag units явно, disk и tmpfs как разные эксперименты; обосновать workload и ресурсы, не обещать заранее полученный результат.

### I23 · P3 · #66: локальные ссылки не открываются на GitHub

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#66](https://github.com/alesplll/opens3-rebac/issues/66).

Ссылки на абсолютные файлы компьютера автора непереносимы. Основная предпосылка issue — отсутствие revocation — соответствует коду; её отклонять не предлагается.

**Переформулировать:** GitHub permalink на исходники и явное разделение refresh revocation/access-token lifetime. Не утверждать, что отзыв refresh автоматически отзывает уже выданный access token.

### I24 · P3 · #17/#23: открытые задачи содержат устаревшую отправную точку

**Статус: ✅ исправлено в body соответствующих открытых issues.**

**Где:** [#17](https://github.com/alesplll/opens3-rebac/issues/17), [#23](https://github.com/alesplll/opens3-rebac/issues/23).

Упомянутого `docker-file.yaml` нет; текущий файл — docker-compose.yml. Для migrator-users уже есть profile/dependency configuration, отличающаяся от старого описания проблемы. Это не доказывает устранение исходного runtime-сбоя без повторного запуска.

**Переформулировать:** актуальные пути, воспроизводимые шаги/ожидаемый результат и пометка «нужна перепроверка». Автоматически закрывать #23 по одному чтению YAML не предлагается.

### I25 · P2 · Закрытые #2–#14/#28 нельзя использовать как актуальный контракт

**Статус: 👀 оставлено владельцу: закрытые исторические issues не переписаны и не переоткрыты.**

**Где:** исторические [#2](https://github.com/alesplll/opens3-rebac/issues/2)–[#14](https://github.com/alesplll/opens3-rebac/issues/14), [#28](https://github.com/alesplll/opens3-rebac/issues/28).

Среди конкретных расхождений: #2 даёт 201 вместо S3 PutObject 200 и предполагает WAL; #8 размещает listing versions на object вместо bucket-level API; #6 связывает WRITER с delete, что расходится с текущими уровнями; #12 вводит direct-priority, которого нет в текущем allow-path; #9 назначает другого producer удаления. Закрытый статус не означает реализации всех этих критериев.

**Переформулировать:** сохранить историю, добавить явные ссылки «superseded by» на согласованный текущий контракт/backlog. Не переносить старые критерии в новые README как подтверждённые возможности.

### I26 · P1 · #50 заявляет готовый sync flow, которому противоречит текущая миграция

**Статус: ✅ C01 исправлен в PR; закрытый исторический body #50 сохранён.**

**Где:** закрытая [#50](https://github.com/alesplll/opens3-rebac/issues/50), её описание как завершённой основы в #52/#65.

В актуальном checkout InsertVersion не заполняет обязательный version_number (C01). Поэтому завершённость issue/наличие unit tests не подтверждает работающий путь с текущей чистой схемой.

**Переформулировать:** не менять исторический смысл закрытия; зафиксировать обнаруженную несовместимость и связать отдельный fix с повторной DB integration проверкой. Не рекламировать текущую запись как проверенно работоспособную до неё.

## Обнаруженные расхождения реализации, которые одной правкой текста не устранить

### C01 · P1 · Metadata InsertVersion не задаёт обязательный version_number

**Статус: ✅ исправлено в PR, добавлен integration test.**

[services/metadata/internal/repository/object/insert_version.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/internal/repository/object/insert_version.go) вставляет object_id/blob_id/size/etag/content_type/state/committed_at. [services/metadata/migrations/20260423000000_init.sql](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/migrations/20260423000000_init.sql) требует `version_number BIGINT NOT NULL` без default; устанавливающего его trigger в миграциях нет. На схеме из этих миграций этот INSERT должен нарушить NOT NULL.

**Для документации:** пометить sync create как имеющий обнаруженный blocker; не выдавать за подтверждённый запуск. **Для отдельного code fix:** конкурентно безопасная выдача version_number и настоящий integration test CreateObjectVersion на миграциях. Вывод статический, Docker-воспроизведения не было.

### C02 · P1 · Metadata delete уже коммитится до незащищённой Kafka-публикации

**Статус: 📌 отражено в обновлённой #65; реализация outbox остаётся отдельной задачей.**

В [services/metadata/internal/service/object/delete.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/internal/service/object/delete.go) и [services/metadata/internal/service/bucket/delete.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/internal/service/bucket/delete.go) сначала удаляются данные, затем отправляется событие. При publish failure возвращается ошибка, но durable outbox record отсутствует. Повтор не обязан восстановить событие об уже удалённой сущности.

**Для документации:** текущая доставка best-effort, возможны orphan data/права; future outbox не описывать как уже реализованный. **Для code fix:** DB state + outbox в одной транзакции, relay и idempotent cleanup.

### C03 · P1 · Полное удаление Metadata теряет ссылки на не-current версии

**Статус: 📌 отражено в обновлённых #56/#59; versioning/GC реализуются отдельно.**

[services/metadata/internal/repository/object/delete.go](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/metadata/internal/repository/object/delete.go) удаляет объект; foreign key в миграции каскадно удаляет versions. Наружу возвращается лишь blob текущей версии. Если у объекта несколько committed versions, события недостаточно для удаления остальных файлов.

**Для документации:** не утверждать полную physical cleanup/versioning lifecycle. **Для code fix:** выбранная семантика удаления и durable перечень всех подлежащих GC blob IDs до потери каталожных ссылок. Это отдельная проблема от delete marker semantics.

### C04 · P1 · Quota может потерять dirty updates после ошибки flush

**Статус: ✅ исправлено в PR, добавлен regression test.**

[services/quota/src/cache/memory.rs](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/src/cache/memory.rs) удаляет dirty marks при snapshot. [services/quota/src/service/quota.rs](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/src/service/quota.rs) при failed flush логирует ошибку и возвращает Ok, но не восстанавливает marks. Без новой мутации этого subject следующий цикл не обязан повторить запись.

**Для документации:** убрать «no data lost»/жёсткое секундное окно. **Для code fix:** надёжная очередь/re-dirty с учётом concurrent updates, ошибка наружу/метрики и fail-retry тест. Отдельно согласовать delete с уже взятым snapshot (D22).

### C05 · P1 · AuthZ invalidation не покрывает зависимости решений

**Статус: 📌 создана #67 с точным failure mode и критериями.**

[services/authz/internal/repositories/cache/invalidation_consumer.py](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/internal/repositories/cache/invalidation_consumer.py) и [services/authz/internal/repositories/neo4j/store.py](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/internal/repositories/neo4j/store.py) формируют/используют hints, недостаточные для всех resource decisions после удаления MEMBER_OF. Есть также окно: старый graph-check заканчивается и кладёт ALLOW после очистки кеша. Отсутствующий consumer в основном запуске усугубляет проблему, но не является единственной причиной.

**Для документации:** current revoke ограничен best-effort/TTL, не immediate. **Для code fix:** dependency-aware invalidation либо epochs/versioned cache, согласованный bounded revoke SLA и race/group-revocation tests.

## Покрытие

### Markdown основного проекта — 23 файла

| Файл | Результат / связанные пункты |
|---|---|
| [README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/README.md) | D01–D04, D08–D09, D28, W19 |
| [GETTING_STARTED.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/GETTING_STARTED.md) | D01, D03 |
| [AGENTS.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/AGENTS.md) | D01, D03; инструкции по стилю не оценивались как факты о runtime |
| [CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/CLAUDE.md) | D01–D03, D08–D14, D28, W10 |
| `docs/auth-users-integration.md` | D04: удалён после проверки по решению владельца |
| [docs/class_diagram.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/class_diagram.md) | D02, D05–D06 |
| [docs/object_diagram.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/object_diagram.md) | D02, D06 |
| [docs/sequence_diagrams.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/sequence_diagrams.md) | D02, D06, D08, W06, W16 |
| [docs/distributed-storage-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/distributed-storage-architecture.md) | W01–W05 |
| [docs/kubernetes-deployment-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/kubernetes-deployment-plan.md) | D28, D30–D32 |
| [docs/storage-fs-architecture.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-fs-architecture.md) | D25–D26, D29, W11 |
| [docs/storage-service-implementation-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/storage-service-implementation-plan.md) | D24–D26, D29, W09 |
| [docs/users-service-unit-test-plan.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/users-service-unit-test-plan.md) | D04: план на конкретную дату, основное содержание соответствует выбранному scope |
| [docs/wiki-audit-2026-04-02.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/docs/wiki-audit-2026-04-02.md) | D04: сохранить исторический статус |
| [e2e/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/e2e/README.md) | Существенных фактических расхождений с helper/Makefile не выявлено; DB запуск не проверялся |
| [services/auth/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/auth/README.md) | D07–D09 |
| [services/users/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/users/README.md) | D03, D05–D06 |
| [services/authz/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/README.md) | D13–D17 |
| [services/authz/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/authz/CLAUDE.md) | D10–D17 |
| [services/quota/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/README.md) | D18–D22 |
| [services/quota/CLAUDE.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/quota/CLAUDE.md) | D18–D22 |
| [services/storage/README.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/README.md) | D23, D25, D28 |
| [services/storage/tests.md](https://github.com/alesplll/opens3-rebac/blob/e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75/services/storage/tests.md) | D27 |

### Опубликованная Wiki — 18 страниц

| Страница | Результат / связанные пункты |
|---|---|
| [Home](https://github.com/alesplll/opens3-rebac/wiki/Home) | D01–D02, D10, W18–W19 |
| [Architecture](https://github.com/alesplll/opens3-rebac/wiki/Architecture) | D01–D02, W01, W03, W06, W10, W17 |
| [Services](https://github.com/alesplll/opens3-rebac/wiki/Services) | D01–D02, D28, W10 |
| [Services-—-Gateway](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Gateway) | D08, D10–D12, D18, W15–W16, I08 |
| [Services-—-Metadata](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Metadata) | W07, W09–W10, W12, C01–C03 |
| [Services-—-AuthZ](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-AuthZ) | D10–D15, W07 |
| [Services-—-Data-Node](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Data-Node) | D25–D26, W03, W09, W12 |
| [Services-—-Utils-—-Kafka](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%94-Utils-%E2%80%94-Kafka) | D14, D29, W06–W09 |
| [Services-‐-Quota](https://github.com/alesplll/opens3-rebac/wiki/Services-%E2%80%90-Quota) | D18–D22, W15 |
| [Architecture-‐-Sequence-Diagrams](https://github.com/alesplll/opens3-rebac/wiki/Architecture-%E2%80%90-Sequence-Diagrams) | D08, D10–D12, W06, W08, W12, W16 |
| [[?]-AuthZ-‐-Cache-Strategy](https://github.com/alesplll/opens3-rebac/wiki/%5B%3F%5D-AuthZ-%E2%80%90-Cache-Strategy) | W13–W14, D15/C05 |
| [[?]-Архитектура-"Metadata-authoritative---pending-finalize"](https://github.com/alesplll/opens3-rebac/wiki/%5B%3F%5D-%D0%90%D1%80%D1%85%D0%B8%D1%82%D0%B5%D0%BA%D1%82%D1%83%D1%80%D0%B0-%22Metadata-authoritative---pending-finalize%22) | D25–D26, D29, W11, W17, I03 |
| [[?]-Архитектура-"Event-driven-commit-and-outbox-inbox"](https://github.com/alesplll/opens3-rebac/wiki/%5B%3F%5D-%D0%90%D1%80%D1%85%D0%B8%D1%82%D0%B5%D0%BA%D1%82%D1%83%D1%80%D0%B0-%22Event-driven-commit-and-outbox-inbox%22) | D26, D29, W11, W17, I03 |
| [Основная-проблема-доставки-и-варианты-ее-решения](https://github.com/alesplll/opens3-rebac/wiki/%D0%9E%D1%81%D0%BD%D0%BE%D0%B2%D0%BD%D0%B0%D1%8F-%D0%BF%D1%80%D0%BE%D0%B1%D0%BB%D0%B5%D0%BC%D0%B0-%D0%B4%D0%BE%D1%81%D1%82%D0%B0%D0%B2%D0%BA%D0%B8-%D0%B8-%D0%B2%D0%B0%D1%80%D0%B8%D0%B0%D0%BD%D1%82%D1%8B-%D0%B5%D0%B5-%D1%80%D0%B5%D1%88%D0%B5%D0%BD%D0%B8%D1%8F) | W01, W11 |
| [Glossary](https://github.com/alesplll/opens3-rebac/wiki/Glossary) | W17 |
| [Roadmap](https://github.com/alesplll/opens3-rebac/wiki/Roadmap) | W18 |
| [Team](https://github.com/alesplll/opens3-rebac/wiki/Team) | W19: требует владельца для подтверждения ролей |
| [_Footer](https://github.com/alesplll/opens3-rebac/wiki/_Footer) | W18 |

### Issues — 37 текстов

Проверены открытые: #17, #23, #36, #49, #51–#66; закрытые: #2–#14, #22, #27, #28, #50. Нумерация пропущенных GitHub номеров не означает пропуск проверки: они не входят в полученную выборку issues (например, могут быть PR).

#22: новых существенных ошибок в тексте не найдено; текущие бейджи отдельно входят в D01. #27: body пустой, предметной проверки требований сделать нельзя. Для закрытых #3–#5, #7, #10–#11, #14/#28 нет отдельного нового доказанного дефекта текста помимо исторического статуса/повторов уже перечисленных проблем. Это не подтверждение реализации всех их acceptance criteria.

## Итог выполнения

- Markdown проекта исправлен в ветке `codex/fix-factual-docs`.
- Все 18 страниц Wiki опубликованы в commit `db3a1c7`.
- Исправлены bodies 20 открытых issues: #17, #23, #36, #49, #51–#66.
- Для неохваченной строгой инвалидации AuthZ создана #67.
- C01 и C04 исправлены кодом и regression tests в PR.
- C02/C03 оставлены реализационными задачами в #65/#56/#59: outbox и version-aware GC требуют межсервисного протокола.
- I25 и состав Team оставлены владельцу для решения; статусы issues не менялись.
