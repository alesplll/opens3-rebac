# Повторная проверка документации — 11 сентября 2026

Проверка завершена 12 сентября 2026; имя файла сохраняет дату начала.

## Область и вывод

Проверена ветка `codex/fix-factual-docs`: исходный checkout аудита `e4b2cfb`,
состояние перед этой проверкой `dbc5585`, изменения README, AGENTS/CLAUDE и планов.
Основания — исходный [аудит](factual-audit-2026-09-10.md), diff, protobuf fields,
handlers/services/repositories, config, миграции, Makefile и Compose.

Первое исправление устранило многие опасные обещания, но часть информации была
удалена вместо актуализации. Последующие commits `85834e3` и `dbc5585` уже вернули
часть архитектурных альтернатив и пояснений; на их основе эта проверка восстановила
практические инструкции. Отметка «✅» в исходном отчёте не всегда означала, что
исправлен каждый пример или выполнен запрошенный тест.

В этой проверке изменена документация; runtime, proto/generated code и имеющиеся
пользовательские изменения `services/authz/proto/generate.sh`/`.DS_Store` сохранены.
Опубликованная Wiki и GitHub issue bodies не перечитывались и не изменялись:
исторические отметки W/I об их публикации здесь независимо не подтверждаются.

## Что восстановлено и исправлено

- Все шесть сервисов имеют README с одинаковыми 12 разделами; Metadata README
  добавлен впервые. Специфические пояснения перенесены в соответствующие разделы,
  а не заменены одинаковым кратким шаблоном.
- Восстановлены gRPC examples, самостоятельный local/Docker startup, полный
  перечень конфигурации, ошибки и отдельные integration-test команды.
- Вернулись карта документации, контекст agent-файлов, тестовые соглашения,
  детали filesystem, backlog Storage, структура будущего Helm chart и topology.
- Сохранены все 16 sequence diagrams и три варианта распределённого data path.
  Исправления внесены внутрь схем, а скрытые HTML warnings заменены видимым статусом.
- Межсервисные policy/event tables восстановлены в
  [Gateway contract plan](gateway-contract-plan.md) с явным статусом предложения.

## Проверка пунктов D01–D32

«Исправлено» ниже относится к тексту рабочего дерева, не к реализации будущих функций.

| Пункт | Результат повторной проверки |
|---|---|
| D01 | Состав сервисов исправлен ранее; теперь добавлены полноценный Metadata README и навигация всех шести сервисов. |
| D02 | Runtime и планы разделены; у class/object diagrams прежние предупреждения были скрыты HTML-комментарием. Сделан видимый статус и исправлено содержимое. |
| D03 | Compose ports/build context сверены. Восстановлены host overrides; Storage staging исправлен с `/data/multipart` на действующий `/data/staging`; исправлен путь `py_kit`. |
| D04 | Даты планов сохранены. Удалённый `auth-users-integration.md` не возвращён: исходный аудит фиксирует решение владельца об удалении. Полезные runbooks восстановлены в README. |
| D05 | Отсутствие Users JWT/TLS/self-only остаётся явно описанным; сохранены CRUD examples и transactional explanation. |
| D06 | В class diagram оставались Users Kafka producer, старый Update и отсутствие UpdatePassword. Исправлены; восстановлены актуальные wire examples. |
| D07 | Нерабочий HTTP startup не возвращён. Вместо удаления всех examples даны рабочие по wire schema grpcurl команды. |
| D08 | В README оставался `ValidateToken → user_id`, хотя response Empty. Исправлены ответ, authorization metadata и тип результата Login в class diagram. |
| D09 | Redis counters/local limiter сохранены; уточнены default кода против dev `.env` и отдельные failure paths чтения/сброса счётчика. |
| D10 | В SD-12 всё ещё был обход «групп и иерархии». Исправлено на MEMBER_OF/direct HAS_PERMISSION; PARENT_OF не выдаёт права. |
| D11 | Публичные enums и иерархия сохранены; добавлены корректные WriteTuple/Check examples, ownership через HAS_PERMISSION + ADMIN. |
| D12 | **Открытое решение policy.** Ранее таблица удалена и пункт отмечен закрытым. Теперь есть единое предложение mapping, включая create/overwrite/multipart; принятие policy и её реализация ещё требуются. |
| D13 | Прежний текст предлагал читателю самому брать поля из proto. Теперь есть примеры, validation cases, allowed=false и отсутствие универсального exception mapping. |
| D14 | README ошибочно добавлял `allowed` в decision payload. Исправлено, восстановлена таблица событий и различие sync produce exception / async delivery error. |
| D15 | Best-effort/TTL сохранены; убрано обещание строгой границы 30s после revoke. Отсчёт TTL начинается от cache set; late check может записать старый ALLOW. |
| D16 | Local setup был неполон: не было venv/install. Добавлены самостоятельные сценарии. Отдельно выявлено, что invalidator entrypoint игнорирует Config и использует localhost defaults. |
| D17 | `internal` остаётся соглашением Python. Вернулись карта слоёв и тестовые patterns из сервисного CLAUDE. |
| D18 | CheckQuota reserve/compensation сохранены; добавлен пример без двойного начисления и оговорка о неизвестном исходе timeout. |
| D19 | Single-writer ограничение сохранено. Проект Kubernetes не предлагает несколько Quota writers при HA Redis. |
| D20 | Неподтверждённые nanoseconds/lock-free не восстановлены. Сохранены объяснение hot path, locks и варианты дальнейшей консистентности. |
| D21 | Default 1000 ms / dev 500 ms и отсутствие AOF сохранены. Добавлены startup load, write-through limits и failure semantics. |
| D22 | DeleteSubject отражён. Восстановлены protoc prerequisite и ignored Redis suite, уточнено различие Cargo cwd/defaults; delete/flush race остаётся ограничением. |
| D23 | Команды Make и go.work объяснение сохранены; generation prerequisites возвращены в Getting Started. |
| D24 | Текущий Storage API отделён от backlog. Восстановлены этапы и acceptance criteria вместо возврата устаревшего API. |
| D25 | Session cleanup/marker flow сохранён и детализирован. Найдена новая ошибка: expected size проверяется service после rename, а не до него; исправлено. |
| D26 | Ограничение file fsync/rename сохранено; добавлены конкретные staging/marker failure windows. |
| D27 | Имена отсутствующих тестов уже исправлены. Инвентарь проверяется по существующим test functions; исторический список не выдаётся за отчёт о запуске. |
| D28 | Standard/custom health разделены, точные команды возвращены для сервисов. Auth/Users custom health не выдумывается. |
| D29 | Вместо одной оговорки добавлена таблица существующих и проектируемых событий, abort, payload и retry. Event contract будущего cleanup остаётся проектом. |
| D30 | Неполный deployment YAML не возвращён как рабочий. Восстановлены chart decomposition и последовательность delivery с prerequisites. |
| D31 | Сохранены selector/labels/topology требования; добавлена иллюстративная таблица трёх failure domains без выдуманного числа pods и готовой HA. |
| D32 | Неподтверждённые цены не возвращены. Drain/crash/partition разделены; сохранены критерии выбора инфраструктуры и проверки восстановления. |

## Пересечения W/I с локальными документами

| Пункты | Проверка в основном репозитории |
|---|---|
| W01–W05 | В distributed-storage плане уже исправлены quorum claims, MultiWriter, logical/local IDs, fencing и сравнение вариантов. Сохранено без нового сокращения. |
| W06–W09, W11 | Схемы больше не называют object-stored готовым backup, способным восстановить Metadata по одному blob_id. Recovery требует intent/operation/version, cleanup consumers явно проектные. |
| W10, I03, I06 | Внешний success barrier после Metadata visibility сохранён; pending overwrite не скрывает current committed version. |
| W12 | Убрано «bucket пуст — файлов на диске нет» из SD-15; physical GC отделён от логической пустоты. |
| W13–W14 | Локальные README не обещают sliding TTL/immediate revoke. Оптимизации отдельной Wiki cache strategy в этой проверке не перепроверялись. |
| W15, I08, I10–I11 | Project quota не названа AWS default; внешний mapping не выдаётся за готовый S3; multipart ETag/part validation отделены от внутреннего Storage. |
| W16 | В SD-10/SD-11 добавлен Check инициатора с ADMIN до mutation, а не только вводная оговорка. |
| W17–W19 | Видимые статусы планов, карта навигации, отсутствие выдуманных масштаба/командных ролей. Состав Team остаётся внешним неподтверждённым сведением. |
| I01–I02, I18–I19 | В локальных plans сохранены operation ID, terminal result, unknown outcome и срок жизни completion marker. |
| Остальные I01–I26 | Bodies issues — отдельные внешние артефакты; эта проверка не означает их повторную валидацию или закрытие. |

## Проверка реализационных C01–C05

| Пункт | Что действительно подтверждено |
|---|---|
| C01 | InsertVersion заполняет version_number, блокирует object row; service вызывает его внутри READ COMMITTED transaction. Unit tests Metadata проходят, integration suites компилируются. Добавленный test проверяет последовательные номера 1/2 напрямую через repository, без service transaction/concurrency. Он **не доказывает** concurrent CreateObjectVersion на реальной БД. |
| C02 | Outbox по-прежнему отсутствует. Publish следует после изменения БД; это явно отражено в Metadata README и таблице событий. |
| C03 | DELETE каскадно удаляет versions и возвращает только current blob. Ограничение полного GC описано; код не выдаётся за законченный lifecycle. |
| C04 | Ошибка flush возвращается наружу и snapshot keys помечаются dirty; значения памяти не заменяются старым snapshot. Regression test проверяет error + повторный dirty snapshot, но **не выполняет** следующий успешный flush и не моделирует concurrent mutation. Rust test здесь не запускался. |
| C05 | Dependency-aware invalidation/epoch нет; README описывает реальные ограничения group revoke и late cache set, а не закрывает проблему изменением текста. |

Изменения C01/C04 направлены на соответствующие дефекты, но прежнее «добавлен
regression test» не следует читать как доказательство всех первоначально
запрошенных сценариев. Для закрытия проверки нужны DB concurrency/service test
для C01 и failed-flush → successful-retry с concurrent update для C04.

## AGENTS.md и CLAUDE.md

AGENTS.md в исходной ветке изменён только по фактам: добавлены Metadata/Quota,
maintenance binaries и команды тестов. Правила minimock, стиля и documentation
не удалялись. Здесь добавлено правило общего README и сохранения полезного содержания.

Корневой CLAUDE.md был существенно сокращён. Часть удаления оправдана: старые
порты, Metadata на Python, OWNER_OF в public API, неверные JWT claims и выдуманные
Kafka consumers возвращать нельзя. Но карта API, сценарии, модель данных и workflow
полезны: они восстановлены в обновлённом виде и связаны с README/планом Gateway.
В AuthZ/Quota CLAUDE восстановлены слои, команды, тестовые patterns и диагностика.

## Выполненные проверки и ограничения

- `go test ./...` в Auth, Users, Metadata и Storage — успешно; Storage включает
  filesystem/component tests.
- Metadata integration suites с `-tags=integration -run '^$'` — компиляция успешна;
  это не запуск проверок PostgreSQL.
- `docker compose --profile services config --quiet` — конфигурация валидна.
- Docker daemon недоступен (socket отсутствует); Compose startup, реальные
  grpcurl calls и Neo4j/Redis/PostgreSQL integration не запускались.
- Rust/Cargo отсутствует в текущем PATH; fmt/clippy/test Quota не запускались.
- Python AuthZ suite не запускался: среда зависимостей сервиса не подготовлена.
- Проверены 27 Markdown-файлов: 180 относительных ссылок, одинаковые 12 разделов
  шести сервисных README, fenced blocks, 69 имён Storage tests и targets Makefile — успешно.
- 23 JSON request bodies из grpcurl examples разобраны через generated protobuf
  descriptors: поля, типы, enum values и имена RPC совпадают с wire schema. Это
  статическая проверка примеров, не вызовы работающего стенда.
- `git diff --check` — успешно.
- PlantUML blocks проверены структурно; renderer в среде не установлен,
  визуальный рендер схем не подтверждён.

Этот отчёт дополняет исходный аудит и уточняет его отметки; он не переписывает
историю публикации Wiki/issues и не объявляет открытые архитектурные решения готовыми.
