# Nginx перед recollect

Обычный вариант — **location на существующем хосте**, не отдельный vhost.

Сниппет: [`nginx/recollect.location.conf`](../nginx/recollect.location.conf).

Публичный URL: `https://<уже-работающий-хост>/recollect/api/collect`.

```text
клиент --> nginx /recollect/ --> 127.0.0.1:8080/recollect/... --> SAYMON
```

Приложение отвечает и на `/api/collect`, и на `/recollect/api/collect`. Nginx **не обрезает** префикс: `proxy_pass` без слэша в конце передаёт полный путь.

## Подготовка

1. Recollect отвечает на `http://127.0.0.1:8080/healthz` и `http://127.0.0.1:8080/recollect/healthz`.
2. Порт 8080 только на localhost:

```yaml
ports:
  - "127.0.0.1:8080:8080"
```

## Вставка в существующий server

```bash
cp /opt/recollect/recollect.location.conf /etc/nginx/snippets/recollect.location.conf
```

Внутри уже работающего `server { ... }`:

```nginx
include /etc/nginx/snippets/recollect.location.conf;
```

Либо скопируйте `location /recollect/` из файла прямо в этот `server`. Соседние `location` не трогайте.

```bash
nginx -t && systemctl reload nginx
```

Проверка:

```bash
curl -s https://<ваш-хост>/recollect/healthz
```

Другой префикс: задайте `HTTP_PREFIX=/другой` у сервиса и поменяйте `location` в сниппете. Несколько префиксов: `HTTP_PREFIX=/recollect,/collect`. Только корень, без префикса: `HTTP_PREFIX=-`.

## Отдельный vhost (необязательно)

Полный сайт: [`nginx/recollect.conf`](../nginx/recollect.conf) — в `conf.d` или `sites-enabled`. В нём замените `server_name`. `default_server` не используется.

## HTTPS

TLS остаётся на существующем хосте. Отдельный сертификат для `/recollect/` не нужен.

Токен лучше передавать заголовком `X-Saymon-Token` или Bearer, не в query.

Примеры запросов — в [usage.md](usage.md).
