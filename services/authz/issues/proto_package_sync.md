# Issue: Синхронизация proto-пакетов — shared vs standalone

## Проблема

В монорепо сосуществуют два proto-файла для AuthZ-сервиса с разными пакетами:

| Файл | package | service |
|---|---|---|
| `proto/authz/v1/authz.proto` (shared, монорепо) | `opens3.authz.v1` | `PermissionService` |
| `services/authz/proto/authz.proto` (standalone, используется реальным кодом) | `rebac.authz.v1` | `PermissionService` |

Gateway будет генерировать стабы из `opens3.authz.v1`, а authz-сервис отдаёт `rebac.authz.v1`.
gRPC routing идёт по имени сервиса (`/package.ServiceName/Method`), поэтому:
- Gateway: вызов `/opens3.authz.v1.PermissionService/Check`
- AuthZ: слушает `/rebac.authz.v1.PermissionService/Check`

Запросы не совпадут — `UNIMPLEMENTED`.

## Варианты решения

### Рекомендуемый: обновить standalone proto (Option A)

1. В `services/authz/proto/authz.proto` сменить пакет:
   ```diff
   - package rebac.authz.v1;
   + package opens3.authz.v1;
   ```
2. Перегенерировать стабы:
   ```bash
   cd services/authz
   bash proto/generate.sh
   ```
3. Проверить, что `entrypoints/server/main.py` регистрирует сервис корректно.

### Альтернатива: authz использует shared proto напрямую (Option B)

Убрать `services/authz/proto/authz.proto`, настроить `generate.sh` на `proto/authz/v1/authz.proto` из корня монорепо.

## Приоритет

Сделать до начала интеграции Gateway ↔ AuthZ (Phase 1 MVP).
