# Nginx перед recollect

Готовый drop-in: [`nginx/recollect.conf`](../nginx/recollect.conf).

Файл кладётся рядом с другими сайтами. `default_server` не используется, соседние vhost не перехватываются.

```text
клиент --> nginx :80/:443 --> 127.0.0.1:8080 recollect --> SAYMON
```

## Подготовка

1. Recollect отвечает на `http://127.0.0.1:8080/healthz`.
2. Порт 8080 не торчит наружу:

```yaml
ports:
  - "127.0.0.1:8080:8080"
```

3. В `recollect.conf` замените `server_name recollect.example.com` на свой хост.

## Включение

Debian/Ubuntu:

```bash
cp nginx/recollect.conf /etc/nginx/sites-available/recollect.conf
ln -s /etc/nginx/sites-available/recollect.conf /etc/nginx/sites-enabled/recollect.conf
nginx -t && systemctl reload nginx
```

Или в общий `conf.d`:

```bash
cp nginx/recollect.conf /etc/nginx/conf.d/recollect.conf
nginx -t && systemctl reload nginx
```

После `deploy.sh` копия лежит в `/opt/recollect/recollect.conf`.

Проверка:

```bash
curl -s -H 'Host: recollect.example.com' http://127.0.0.1/healthz
```

## HTTPS

В том же файле есть закомментированный `server` на 443. Раскомментируйте его, укажите сертификаты и в блоке `:80` добавьте `return 301 https://$host$request_uri;`.

Let's Encrypt:

```bash
certbot --nginx -d recollect.example.com
```

После certbot проверьте, что `proxy_read_timeout 120s` на месте.

Токен лучше передавать заголовком `X-Saymon-Token`: формат `recollect_access` не пишет query string в лог.

Примеры запросов — в [usage.md](usage.md).
