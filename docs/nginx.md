# Nginx перед recollect

Nginx принимает внешний HTTP/HTTPS и проксирует на recollect, который слушает только localhost.  
Сборку см. в [build.md](build.md), установку сервиса — в [install.md](install.md).

Схема:

```text
клиент --> :443 nginx --> 127.0.0.1:8080 recollect --> SAYMON
```

## Что подготовить на сервере

1. Recollect уже запущен и отвечает на `http://127.0.0.1:8080/healthz`.
2. Порт `8080` не торчит наружу. Для Compose:

```yaml
ports:
  - "127.0.0.1:8080:8080"
```

Для systemd:

```ini
Environment=LISTEN_ADDR=127.0.0.1:8080
```

3. Пакет nginx установлен.

## Конфиг сайта

Файл `/etc/nginx/sites-available/recollect` (Debian/Ubuntu) или `/etc/nginx/conf.d/recollect.conf`.

Подставьте свой `server_name` и пути к сертификатам.

```nginx
log_format recollect '$remote_addr - $remote_user [$time_local] '
                     '"$request_method $uri" $status $body_bytes_sent '
                     '"$http_referer" "$http_user_agent" rt=$request_time';

upstream recollect {
    server 127.0.0.1:8080;
    keepalive 8;
}

server {
    listen 80;
    listen [::]:80;
    server_name recollect.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name recollect.example.com;

    ssl_certificate     /etc/ssl/certs/recollect.crt;
    ssl_certificate_key /etc/ssl/private/recollect.key;

    access_log /var/log/nginx/recollect.access.log recollect;
    error_log  /var/log/nginx/recollect.error.log;

    # запрос к SAYMON может идти десятки секунд
    proxy_connect_timeout 5s;
    proxy_send_timeout    120s;
    proxy_read_timeout    120s;

    location / {
        proxy_pass http://recollect;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Saymon-Token $http_x_saymon_token;
        proxy_set_header Authorization $http_authorization;
    }
}
```

В `log_format` нет `$args` и `$request`: токен из query не пишется в access-лог. Клиентам лучше передавать токен заголовком `X-Saymon-Token`, а не `?token=`.

Включить сайт и проверить:

```bash
ln -s /etc/nginx/sites-available/recollect /etc/nginx/sites-enabled/recollect
nginx -t
systemctl reload nginx
curl -s https://recollect.example.com/healthz
```

## Только HTTP (без TLS)

Если TLS закрывает другой балансировщик:

```nginx
log_format recollect '$remote_addr - $remote_user [$time_local] '
                     '"$request_method $uri" $status $body_bytes_sent '
                     'rt=$request_time';

upstream recollect {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    listen [::]:80;
    server_name recollect.example.com;

    access_log /var/log/nginx/recollect.access.log recollect;

    proxy_connect_timeout 5s;
    proxy_send_timeout    120s;
    proxy_read_timeout    120s;

    location / {
        proxy_pass http://recollect;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Saymon-Token $http_x_saymon_token;
        proxy_set_header Authorization $http_authorization;
    }
}
```

## Let's Encrypt

Если домен смотрит на этот сервер:

```bash
apt install certbot python3-certbot-nginx
certbot --nginx -d recollect.example.com
```

Certbot сам допишет `ssl_certificate`. Таймауты `proxy_read_timeout` после выпуска сертификата стоит проверить — certbot их не всегда сохраняет.

## Проверка через прокси

```bash
curl -s https://recollect.example.com/healthz

curl -G 'https://recollect.example.com/api/collect' \
  -H 'X-Saymon-Token: 682a3631-cfb2-49dd-b09f-858280a240dd' \
  --data-urlencode 'host=saymon.example.com' \
  --data-urlencode 'object_id=6a61ae4562e391eba8d3edbb' \
  --data-urlencode 'metrics=message.I' \
  --data-urlencode 'aggregates=avg' \
  --data-urlencode 'depth=12h'
```

Подробности API — в [usage.md](usage.md).
