# Getting Started

Этот документ нужен для самого простого сценария:

- у вас установлен только Docker
- вы хотите поднять локально всё, что уже реализовано в проекте
- вы не хотите ставить Go, Python, `make`, `grpcurl` и другие утилиты

## Что важно заранее

Все команды ниже нужно запускать **из корня репозитория**.

То есть сначала нужно перейти в директорию проекта:

```bash
cd /путь/до/opens3-rebac
```

Пример:

```bash
cd ~/projects/opens3-rebac
```

Если запускать команды не из корня репозитория, `docker compose` не найдёт `docker-compose.yml`, `env_file` и Docker build context.

## Что должно быть установлено

Достаточно:

- Docker
- Docker Compose plugin

Проверка:

```bash
docker --version
docker compose version
```

## Что поднимется

На текущем этапе реально поднимаются:

- `users`
- `auth`
- `authz`
- `storage`

Вместе с инфраструктурой:

- `postgres-users`
- `postgres-metadata`
- `redis`
- `neo4j`
- `zookeeper`
- `kafka`
- `migrator-users`

Пока не реализованы как рабочие сервисы:

- `gateway`
- `metadata`

Поэтому полного S3 flow ещё нет. Этот запуск нужен для локальной разработки и проверки уже существующих сервисов.

## Самый короткий путь

Из корня репозитория выполните:

```bash
docker compose --profile services up --build -d
```

Эта команда:

- поднимет инфраструктуру
- соберёт локальные образы сервисов
- запустит все сервисы из профиля `services`
- дождётся зависимостей `auth`, `users`, `authz`, `storage` и старта контейнера `metadata` перед стартом `gateway`

## Как проверить, что всё поднялось

Проверьте список контейнеров:

```bash
docker compose ps
```

Ожидаемо должны быть в состоянии `Up` или `Exited (0)` для одноразового мигратора:

- `postgres-users`
- `postgres-metadata`
- `redis`
- `neo4j`
- `zookeeper`
- `kafka`
- `migrator-users`
- `users`
- `auth`
- `authz`
- `storage`

## Полезные команды

Посмотреть статус:

```bash
docker compose ps
```

Посмотреть логи всех сервисов:

```bash
docker compose logs -f
```

Посмотреть логи одного сервиса:

```bash
docker compose logs -f users
docker compose logs -f auth
docker compose logs -f authz
docker compose logs -f storage
```

Остановить всё:

```bash
docker compose --profile services down
```

Остановить всё и удалить volume-данные:

```bash
docker compose --profile services down -v
```

Пересобрать и поднять заново:

```bash
docker compose --profile services up --build -d
```

## Порты

Основные локальные порты:

- `users` → `localhost:50054`
- `auth` → `localhost:50050`
- `authz` → `localhost:50051`
- `storage` → `localhost:50053`
- `postgres-users` → `localhost:5432`
- `postgres-metadata` → `localhost:5433`
- `redis` → `localhost:6379`
- `kafka` → `localhost:9092`
- `neo4j http` → `localhost:7474`
- `neo4j bolt` → `localhost:7687`

Neo4j Browser:

- URL: `http://localhost:7474`
- login: `neo4j`
- password: `password123`

## Что делать, если что-то не поднялось

1. Убедиться, что вы находитесь в корне репозитория.
2. Проверить `docker compose ps`.
3. Посмотреть логи проблемного сервиса:

```bash
docker compose logs -f <service-name>
```

Чаще всего проблемы будут такие:

- уже занят локальный порт
- Docker daemon не запущен
- не хватает ресурсов Docker
- один из зависимых контейнеров не стал healthy

## Если `make` установлен

Можно использовать короткие команды-обёртки:

```bash
make up-services
make down
make down-volumes
make rebuild
```

Но для первого запуска `make` не нужен: все основные шаги выше работают напрямую через `docker compose`.
