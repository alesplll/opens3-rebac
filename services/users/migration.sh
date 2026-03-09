#!/bin/bash
set -e

if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

DB_HOST=${PG_HOST:-localhost}
DB_PORT=${PG_PORT_INNER:-5432}
DB_NAME=${PG_DATABASE_NAME:-postgres}
DB_USER=${PG_USER:-postgres}
DB_PASSWORD=${PG_PASSWORD:-}
MIGRATION_DIR=${MIGRATION_DIR:-migrations}

MIGRATION_DSN="host=${DB_HOST} port=${DB_PORT} dbname=${DB_NAME} user=${DB_USER} password=${DB_PASSWORD} sslmode=disable"

echo "Running migrations with DSN: ${MIGRATION_DSN}"
goose -dir "${MIGRATION_DIR}" postgres "${MIGRATION_DSN}" up -v
