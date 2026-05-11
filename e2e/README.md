# E2E Test Kit

`e2e/` это общий пакет с test infrastructure для интеграционных и будущих end-to-end тестов по всему монорепо.

Сейчас он решает две задачи:

- даёт общий `BaseSuite` с конфигом, контекстом и общими helper methods
- даёт `BasePostgresSuite` для тестов, которым нужна реальная PostgreSQL база с миграциями и полным cleanup перед каждым тестом

## Структура

- `suite/base_suite.go` — базовый `testify/suite` с загрузкой `e2e/.env`, `Context()`, `Config()`, `FixtureUUID()`
- `suite/base_postgres_suite.go` — база для Postgres integration tests: connect, reset schema, apply migrations, cleanup tables
- `suite/config/` — загрузка `e2e/.env` и резолв путей от корня репозитория
- `suite/postgres/` — низкоуровневые Postgres helpers
- `.env` — локальный конфиг для e2e/integration окружения

## Базовый подход

- тесты используют реальную PostgreSQL в Docker
- тесты идут последовательно, без параллельного запуска
- таблицы чистятся перед каждым тестом, не после
- тестовые данные должны создаваться явно внутри теста или через `mustRecreate*` helpers
- пути до `.env` и миграций считаются от корня репозитория, а не от текущего каталога запуска

## Как использовать

Если тесту нужен только общий конфиг и helper methods:

```go
type MySuite struct {
    suite.BaseSuite
}
```

Если тесту нужен реальный Postgres:

```go
type MySuite struct {
    suite.BasePostgresSuite
}

func (s *MySuite) SetupSuite() {
    s.SetupPostgresSuite(suite.PostgresSuiteOptions{
        MigrationDir:  s.MustServiceMigrationDir("metadata"),
        CleanupTables: []string{"versions", "objects", "buckets"},
        ResolvePGConfig: func(cfg *config.Config) config.PGConfig {
            return cfg.MetadataPG
        },
    })
}
```

## Локальный запуск

Поднять e2e Postgres:

```bash
make up-e2e
```

Запустить metadata integration tests:

```bash
make test-metadata-integration
```

Поднять контейнер и сразу прогнать тесты одним target:

```bash
make test-metadata-integration-local
```

Остановить e2e окружение:

```bash
make down-e2e
```

## Правила для новых suite

- переиспользуйте `BaseSuite` и `BasePostgresSuite`, не дублируйте bootstrap код по сервисам
- миграции берите через `MustServiceMigrationDir("<service>")`
- для строковых fixture values удобно использовать `suite.T().Name()`
- для UUID используйте `FixtureUUID(...)`, чтобы значения были детерминированными
- helper methods вида `mustRecreate*` должны именно пересоздавать сущность, а не просто делать `INSERT`
