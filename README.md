# recollect

Сервис пересчитывает статистику родительского объекта SAYMON по истории дочерних метрик.

Сборка и установка разнесены: образ или бинарник собираются в одном месте, на целевой хост копируется только артефакт.

## Документация

- [Сборка](docs/build.md) — `./scripts/build-arm.sh` или `./scripts/build-amd.sh`
- [Установка](docs/install.md) — `./scripts/deploy.sh архив.tar.gz` на целевой машине
- [Использование](docs/usage.md) — API и примеры запросов
- [Nginx](docs/nginx.md) — HTTPS и reverse proxy
