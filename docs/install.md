# Установка

Установка выполняется на **целевой машине**, отдельно от сборки.  
Нужен архив со [сборочной машины](build.md): `dist/recollect-*-linux-arm64.tar.gz` или `dist/recollect-*-linux-amd64.tar.gz` — под архитектуру целевого сервера.

Основной путь — скрипт `deploy.sh` (он же лежит внутри архива).

## Требования на целевой машине

- Docker
- Docker Compose plugin (`docker compose`) или `docker-compose`
- `curl` или `wget` для проверки `/healthz`

Исходный код и Go не нужны.

## Запуск из архива

Скопируйте архив на сервер и выполните скрипт из репозитория:

```bash
./scripts/deploy.sh ./recollect-1.0.0-linux-amd64.tar.gz
```

Для ARM64 подставьте `linux-arm64` в имени архива.

Или без репозитория — только архив:

```bash
tar -xzf recollect-1.0.0-linux-amd64.tar.gz
cd recollect-1.0.0-linux-amd64
./deploy.sh
```

По умолчанию всё ставится в `/opt/recollect`. Каталог можно сменить:

```bash
INSTALL_DIR=/srv/recollect ./scripts/deploy.sh ./recollect-1.0.0-linux-amd64.tar.gz
```

Скрипт загружает образ, копирует compose-конфиг, поднимает контейнер и проверяет `http://127.0.0.1:8080/healthz`.

Если `.env` уже есть, он не затирается: обновляется только `RECOLLECT_IMAGE`.

## Настройка после установки

Файл `/opt/recollect/.env`:

```dotenv
RECOLLECT_IMAGE=recollect:1.0.0
LISTEN_ADDR=:8080
HTTP_TIMEOUT=30s
SAYMON_CONCURRENCY=8
SAYMON_TLS_INSECURE=false
```

Самоподписанный сертификат SAYMON:

```dotenv
SAYMON_TLS_INSECURE=true
```

Если сервис будет за [nginx](nginx.md) на той же машине, ограничьте порт localhost в `/opt/recollect/docker-compose.yml`:

```yaml
ports:
  - "127.0.0.1:8080:8080"
```

Затем:

```bash
cd /opt/recollect
docker compose up -d
```

Конфиг nginx после установки: `/opt/recollect/recollect.location.conf` — вставка в существующий `server`. Отдельный vhost: `recollect.conf`. См. [nginx.md](nginx.md).


## Обновление

Повторите `deploy.sh` с новым архивом. Существующий `.env` сохранится, подтянется новый тег образа.

Логи:

```bash
cd /opt/recollect
docker compose logs -f recollect
```

Остановка:

```bash
cd /opt/recollect
docker compose down
```

## systemd (бинарник, без Docker)

Если собрали бинарник, а не образ:

```bash
install -m 0755 recollect /usr/local/bin/recollect
useradd --system --no-create-home --shell /usr/sbin/nologin recollect || true
```

Юнит `/etc/systemd/system/recollect.service`:

```ini
[Unit]
Description=recollect SAYMON metrics aggregator
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=recollect
Group=recollect
Environment=LISTEN_ADDR=127.0.0.1:8080
Environment=HTTP_TIMEOUT=30s
Environment=SAYMON_CONCURRENCY=8
Environment=SAYMON_TLS_INSECURE=false
ExecStart=/usr/local/bin/recollect
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now recollect
curl -s http://127.0.0.1:8080/healthz
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `LISTEN_ADDR` | `:8080` | Адрес HTTP-сервера |
| `HTTP_TIMEOUT` | `30s` | Таймаут одного запроса к SAYMON |
| `SAYMON_CONCURRENCY` | `8` | Параллельные запросы истории |
| `SAYMON_TLS_INSECURE` | `false` | Не проверять TLS-сертификат SAYMON |
| `HTTP_PREFIX` | `/recollect` | Дополнительный URL-префикс. Корень `/` всегда доступен. `-` — только корень |

Дальше — [использование](usage.md) и [nginx](nginx.md).
