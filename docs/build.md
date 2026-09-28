# Сборка

Сборка выполняется на **сборочной машине**. На ней нужен исходный код и Docker (или Go 1.23+).  
На машину установки исходники и компилятор **не** нужны: туда передаётся только артефакт.

Артефакты:

- Docker-образ `recollect:<tag>` — основной способ
- бинарник `recollect` — если сервис запускается без Docker

## Требования на сборочной машине

- Docker (для образа)
- или Go 1.23+ (для бинарника)
- сеть для скачивания базовых образов `golang` и `alpine` при первой сборке

## Сборка образа

Из корня репозитория:

```bash
docker build -t recollect:1.0.0 .
docker tag recollect:1.0.0 recollect:local
```

Проверка образа:

```bash
docker run --rm -p 8080:8080 recollect:1.0.0
curl -s http://127.0.0.1:8080/healthz
```

Ожидается `{"status":"ok"}`.

## Выгрузка образа для установки

Образ сохраняется в файл и копируется на целевой хост.

```bash
docker save recollect:1.0.0 | gzip > recollect-1.0.0.tar.gz
```

Вместе с образом на установку передайте `docker-compose.yml` из корня репозитория.

## Сборка бинарника

Linux, статическая линковка (подходит для systemd без Docker):

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o dist/recollect ./cmd/recollect
```

Для ARM64:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o dist/recollect ./cmd/recollect
```

Проверка на той же архитектуре:

```bash
go test ./...
```

Файл `dist/recollect` копируется на машину установки.

## Что передавать на установку

| Артефакт | Куда | Зачем |
| --- | --- | --- |
| `recollect-1.0.0.tar.gz` | сервер приложения | загрузка образа |
| `docker-compose.yml` | сервер приложения | запуск контейнера |
| `dist/recollect` | сервер приложения | запуск без Docker |
| `docs/install.md` | опционально | инструкция на месте |

Дальше — [установка](install.md).
