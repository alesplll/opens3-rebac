# Документация OpenS3-ReBAC

## С чего начать

[Обзор проекта](../README.md) описывает назначение и состояние реализации;
[Getting Started](../GETTING_STARTED.md) — запуск стенда, адреса и диагностику.

## Сервисы

| Сервис | Ответственность | Документация |
|---|---|---|
| Auth | credentials через Users, выпуск и проверка JWT | [README](../services/auth/README.md) |
| Users | учётные записи и password hashes | [README](../services/users/README.md) |
| AuthZ | ReBAC graph и decision cache | [README](../services/authz/README.md) |
| Metadata | buckets, keys и версии | [README](../services/metadata/README.md) |
| Storage | blob bytes и multipart | [README](../services/storage/README.md) |
| Quota | резервирование и учёт квот | [README](../services/quota/README.md) |

У всех сервисных README одинаковый порядок: назначение → архитектура → структура →
API → данные/сценарии → конфигурация → запуск → примеры → диагностика → разработка/тесты →
ограничения → ссылки. Специфические подробности сохраняются внутри этих разделов.
Корневой README, эта карта и E2E README описывают другие сущности и имеют свою структуру.

## Текущее устройство и разработка

- [Filesystem Storage](storage-fs-architecture.md): фактические пути и commit.
- [Storage test inventory](../services/storage/tests.md): существующие проверки.
- [E2E kit](../e2e/README.md): общий test harness и отдельная PostgreSQL.
- [AGENTS.md](../AGENTS.md): правила изменений, тестов и документации.
- [CLAUDE.md](../CLAUDE.md): контекст сервисов и межсервисные ограничения.

## Проектируемая архитектура

Это проекты следующих этапов, а не готовый runtime:

- [Storage implementation plan](storage-service-implementation-plan.md).
- [Репликация, placement и варианты data path](distributed-storage-architecture.md).
- [Kubernetes deployment](kubernetes-deployment-plan.md).
- [Модель классов](class_diagram.md), [модель объектов](object_diagram.md),
  [последовательности операций](sequence_diagrams.md) — PlantUML с пояснениями статуса.
- [Gateway: проект политики доступа и событий](gateway-contract-plan.md).

Исходные подробные версии планов доступны в Git по commit
`e4b2cfbd94e965c04443dd7ff6d482cdd67b7e75`. Они содержат устаревшие контракты;
для разработки используйте актуальные тексты, а историю — для восстановления мотивации.

## Аудиты и исторические планы

- [Проверка исправлений от 11 сентября](documentation-review-2026-09-11.md):
  что осталось после первого исправления и что восстановлено.
- [Wiki audit от 2 апреля](wiki-audit-2026-04-02.md): исторический снимок.
- [План Users unit tests](users-service-unit-test-plan.md): датированный scope.
- [Опубликованная Wiki](https://github.com/alesplll/opens3-rebac/wiki): отдельный
  репозиторий; её публикация не обновляется автоматически изменением `docs/`.
