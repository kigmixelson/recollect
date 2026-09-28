# Сборка

Сборка выполняется на **сборочной машине**. На целевую машину исходники не копируются — только архив.

Архив содержит Docker-образ нужной архитектуры, `docker-compose.yml`, `.env` и `deploy.sh`.

## Требования на сборочной машине

- Docker (BuildKit)
- `curl`/`git` по желанию
- Go 1.23+ опционально: если `go` нет в PATH, тесты идут в контейнере `golang:1.23-alpine`

Компилятор работает на архитектуре сборочной машины, qemu не требуется.

## Сборка скриптом

ARM64:

```bash
./scripts/build-arm.sh
```

Результат: `dist/recollect-<версия>-linux-arm64.tar.gz`

AMD64 (x86_64):

```bash
./scripts/build-amd.sh
```

Результат: `dist/recollect-<версия>-linux-amd64.tar.gz`

Версию можно задать явно:

```bash
VERSION=1.0.0 ./scripts/build-amd.sh
```

Архитектуру целевой машины и архива нужно брать одну и ту же.

## Что внутри архива

- `image.tar.gz` — `docker save` образа `recollect:<версия>-<arch>`
- `docker-compose.yml`
- `.env` с `RECOLLECT_IMAGE=recollect:<версия>-<arch>`
- `deploy.sh`
- `recollect.conf` — отдельный vhost для nginx (необязательно)
- `recollect.location.conf` — `location /recollect/` для существующего хоста
- `VERSION`

Этот файл копируется на целевую машину. Дальше — [установка](install.md).

## Ручная сборка

```bash
docker build --platform linux/amd64 --build-arg TARGETARCH=amd64 -t recollect:1.0.0-amd64 .
docker save recollect:1.0.0-amd64 | gzip > recollect-1.0.0-linux-amd64.tar.gz
```

Бинарник без Docker:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o dist/recollect ./cmd/recollect
```
