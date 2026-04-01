# План покрытия unit-тестами сервисного слоя `users`

Дата: 2026-04-02
Ветка: `test/users/basic_tests`

## Цель

Покрыть unit-тестами сервисный слой `services/users/internal/service/user` без интеграции с БД, gRPC handler-ами и внешней инфраструктурой.

В текущем объёме покрываем только публичные методы `UserService`:

- `Create`
- `Get`
- `Update`
- `UpdatePassword`
- `Delete`
- `ValidateCredentials`

## Границы тестирования

Тестируем:

- бизнес-валидацию на уровне service
- корректную передачу аргументов в `UserRepository`
- корректную обработку ошибок репозитория
- транзакционное поведение `UpdatePassword`
- поведение сервиса при валидных и невалидных входных данных

Не тестируем:

- SQL и реализацию repository
- handler-слой
- интеграцию с PostgreSQL
- внутренности `bcrypt`, `tracing`, `logger`, `validator` библиотек
- e2e / integration сценарии

## Подход

- использовать только unit tests
- использовать существующие моки из `services/users/pkg/mocks`
- писать table-driven tests в `services/users/internal/service/user/tests`
- для каждого `t.Run` создавать отдельный `minimock.Controller`, чтобы не ловить гонки и flaky-поведение

## План по методам

### 1. `Create`

Расширить уже существующий тестовый набор:

- success case
- ошибка `repo.Create`
- пустое имя
- пустой email
- невалидный email
- пустой пароль
- несовпадение `password` и `passwordConfirm`
- слишком короткий пароль

Дополнительно проверить:

- `repo.Create` не вызывается при ошибке валидации
- в success case пароль передаётся в репозиторий уже в захешированном виде
- в success case `createdAt` не нулевой

### 2. `Get`

Добавить отдельные unit-тесты:

- success case: сервис возвращает пользователя из репозитория без изменений
- ошибка `repo.Get` пробрасывается наверх

### 3. `Update`

Добавить unit-тесты:

- success case с валидными `name` и `email`
- ошибка `repo.Update`
- пустой `name`
- пустой `email`
- невалидный email

Отдельно проверить:

- при ошибке валидации `repo.Update` не вызывается
- при success в репозиторий уходят те же значения `userID`, `name`, `email`

### 4. `UpdatePassword`

Добавить unit-тесты на транзакционный сценарий:

- success case: в рамках `ReadCommitted` вызываются `UpdatePassword` и `LogPassword`
- ошибка `repo.UpdatePassword`
- ошибка `repo.LogPassword`
- ошибка `txManager.ReadCommitted`
- пустой пароль
- несовпадение `password` и `passwordConfirm`
- слишком короткий пароль
- отсутствие IP в контексте

Отдельно проверить:

- при ошибке валидации транзакция не стартует
- при ошибке `UpdatePassword` вызов `LogPassword` не выполняется
- в `UpdatePassword` в репозиторий уходит не исходный пароль, а hash

### 5. `Delete`

Добавить unit-тесты:

- success case
- ошибка `repo.Delete`

### 6. `ValidateCredentials`

Добавить unit-тесты:

- success case: hash совпадает, сервис возвращает `(true, userID)`
- пользователь не найден: сервис возвращает `(false, "")`
- ошибка репозитория, отличная от `ErrUserNotFound`: сервис возвращает `(false, "")`
- неверный пароль: сервис возвращает `(false, "")`

## Порядок реализации

Предлагаемый порядок, чтобы быстро получить базовое покрытие без лишней связности:

1. `Get`
2. `Delete`
3. `Update`
4. `UpdatePassword`
5. `ValidateCredentials`
6. расширение `Create`

## Критерий готовности

Задачу считаем завершённой, когда:

- все публичные методы `UserService` покрыты unit-тестами
- happy-path и основные error-path покрыты для каждого метода
- тесты стабильно проходят локально

Минимальная проверка:

```bash
go test ./services/users/internal/service/user/tests -v
```
