# Сборка

Сборка выполняется на **сборочной машине**. На целевую машину исходники не копируются — только архив из `scripts/build-arm.sh`.

Архив содержит linux/arm64 Docker-образ, `docker-compose.yml`, `.env` и `deploy.sh`.

## Требования на сборочной машине

- Docker (BuildKit)
- `curl`/`git` по желанию
- Go 1.23+ опционально: если `go` нет в PATH, тесты идут в контейнере `golang:1.23-alpine`

Скрипт собирает **linux/arm64**. Запускайте его на ARM-хосте или на Docker с эмуляцией `linux/arm64`.

## Сборка скриптом

Из корня репозитория:

```bash
./scripts/build-arm.sh
```

Результат: `dist/recollect-<версия>-linux-arm64.tar.gz`

Версию можно задать явно:

```bash
VERSION=1.0.0 ./scripts/build-arm.sh
```

## Что внутри архива

- `image.tar.gz` — `docker save` образа `recollect:<версия>`
- `docker-compose.yml`
- `.env` с `RECOLLECT_IMAGE=recollect:<версия>`
- `deploy.sh`
- `VERSION`

Этот файл копируется на целевую машину. Дальше — [установка](install.md).

## Ручная сборка

```bash
docker build --platform linux/arm64 -t recollect:1.0.0 .
docker save recollect:1.0.0 | gzip > recollect-1.0.0.tar.gz
```

Бинарник без Docker:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o dist/recollect ./cmd/recollect
```
