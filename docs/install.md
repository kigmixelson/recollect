# Установка

Установка выполняется на **целевой машине**, отдельно от сборки.  
Исходный код и Go здесь не нужны. Нужен артефакт со [сборочной машины](build.md).

Два варианта:

1. Docker Compose — основной
2. systemd и бинарник — если контейнеры не используются

## Docker Compose

Нужны Docker и Docker Compose plugin (или `docker-compose`).

### 1. Загрузить образ

```bash
gunzip -c recollect-1.0.0.tar.gz | docker load
docker image ls recollect
```

Если тег в файле `docker-compose.yml` другой, задайте его явно:

```bash
docker tag recollect:1.0.0 recollect:local
```

или:

```bash
export RECOLLECT_IMAGE=recollect:1.0.0
```

### 2. Положить compose-файл

Скопируйте `docker-compose.yml` в каталог запуска, например `/opt/recollect`.

```bash
mkdir -p /opt/recollect
cp docker-compose.yml /opt/recollect/
cd /opt/recollect
```

### 3. Настроить окружение

Создайте `/opt/recollect/.env` при необходимости:

```dotenv
RECOLLECT_IMAGE=recollect:1.0.0
LISTEN_ADDR=:8080
HTTP_TIMEOUT=30s
SAYMON_CONCURRENCY=8
SAYMON_TLS_INSECURE=false
```

Если у SAYMON самоподписанный сертификат:

```dotenv
SAYMON_TLS_INSECURE=true
```

Если сервис будет за nginx на той же машине, публиковать порт наружу не нужно. В `docker-compose.yml` замените проброс порта на localhost:

```yaml
ports:
  - "127.0.0.1:8080:8080"
```

### 4. Запустить

```bash
cd /opt/recollect
docker compose up -d
docker compose ps
curl -s http://127.0.0.1:8080/healthz
```

Логи:

```bash
docker compose logs -f recollect
```

Обновление с новой сборки: загрузить новый tar, `docker compose up -d`.

Остановка:

```bash
docker compose down
```

## systemd (бинарник)

Нужен файл `recollect`, собранный под архитектуру сервера.

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
systemctl status recollect
curl -s http://127.0.0.1:8080/healthz
```

Для доступа снаружи либо откройте порт в файрволе, либо поставьте [nginx](nginx.md).

## Переменные окружения

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `LISTEN_ADDR` | `:8080` | Адрес HTTP-сервера |
| `HTTP_TIMEOUT` | `30s` | Таймаут одного запроса к SAYMON |
| `SAYMON_CONCURRENCY` | `8` | Параллельные запросы истории |
| `SAYMON_TLS_INSECURE` | `false` | Не проверять TLS-сертификат SAYMON |

Дальше — [использование](usage.md) и [nginx](nginx.md).
