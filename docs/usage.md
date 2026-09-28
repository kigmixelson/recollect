# Использование

Сервис слушает HTTP. После [установки](install.md) проверьте liveness и вызывайте расчёт агрегатов.

## Эндпоинты

| Метод | Путь | Назначение |
| --- | --- | --- |
| `GET` | `/healthz` | Проверка, что процесс жив |
| `GET` | `/api/collect` | Расчёт, параметры в query |
| `POST` | `/api/collect` | Расчёт, параметры в JSON-теле — удобно для HTTP-проверки SAYMON |

## Как считается ответ

1. Сервис запрашивает `GET /node/api/objects/{id}/children` на указанном SAYMON-хосте.
2. Метрики из запроса накладываются на `metrics_cache` каждого дочернего объекта.
3. Для совпавших метрик запрашивается история `GET /node/api/objects/{childId}/history` на глубину от текущего момента.
4. Возвращаются агрегаты по каждой совпавшей метрике.

Дети без совпадений попадают в `skipped`.

## Параметры `/api/collect`

`GET` — query string. `POST` — JSON в теле. Заголовки перекрывают токен и хост.

Значения с пробелами в GET кодируйте (`curl -G --data-urlencode`).

| Параметр | Обязательный | Пример | Описание |
| --- | --- | --- | --- |
| `host` | да | `saymon.example.com` | Имя SAYMON-хоста. Схема необязательна, по умолчанию `https` |
| `token` | да | `682a3631-...` | Ключ доступа SAYMON |
| `object_id` | да | `6a61ae4562e391eba8d3edbb` | ID родительского объекта |
| `metrics` | да | `message.I` | Метрики: повторять параметр или перечислить через запятую |
| `aggregates` | да | `avg` | Агрегаты: можно несколько |
| `depth` | да | `12h` | Глубина окна от текущего момента |

Синонимы query-параметров:

- хост: `host`, `hostname`, `saymon_host`, заголовок `X-Saymon-Host`
- токен: `token`, `auth-token`, `api-token`, заголовок `X-Saymon-Token` или `Authorization: Bearer ...`
- объект: `object_id`, `objectId`, `object`, `id`
- метрики: `metrics`, `metric`, `metrics[]`
- агрегаты: `aggregates`, `aggregate`, `aggs`, `aggregates[]`
- глубина: `depth`, `window`, `period`

Токен в query попадает в access-логи. Для продакшена лучше заголовок `X-Saymon-Token` или `Authorization: Bearer`, см. [nginx](nginx.md).

## HTTP-проверка SAYMON

В сенсоре **HTTP request** лучше `POST` и JSON в **Request body**: URL короткий, массивы метрик читаются, токен не светится в строке адреса.

| Поле в форме | Значение |
| --- | --- |
| Request type | `POST` |
| URL | `http://<recollect-host>/api/collect` |
| HTTP headers | `Content-Type` = `application/json` |
| Request body | JSON ниже |
| HTTP Auth | `Bearer`, токен в поле username |
| Timeout | `120000` (мс), запросы к истории могут быть долгими |
| Send response body | включено |
| Response format | JSON, если есть в списке, иначе Autodetect |

```json
{
  "host": "saymon.example.com",
  "object_id": "6a61ae4562e391eba8d3edbb",
  "metrics": ["message.I", "message.Temp"],
  "aggregates": ["avg", "min", "max", "sum", "dev"],
  "depth": "12h"
}
```

Токен в JSON тоже можно (`"token": "..."`), но в этой форме удобнее Bearer.

`GET` с длинным query тоже работает, его имеет смысл только для разовых проверок из curl.

## Агрегаты

| Значение в ответе | Что считает | Синонимы во входе |
| --- | --- | --- |
| `avg` | среднее | `average`, `mean`, `среднее` |
| `min` | минимум | `minimum`, `минимум` |
| `max` | максимум | `maximum`, `максимум` |
| `sum` | сумма | `total`, `сумма` |
| `dev` | стандартное отклонение | `stddev`, `девиация`, `дивиация` |

На SAYMON уходит `downsample=all-avg|all-min|all-max|all-sum|all-dev`. Если точек несколько, агрегат дополнительно считается в сервисе.

## Глубина

Примеры: `12h`, `24h`, `3m`, `90s`, `1d`, `12 hours`, `12 часов`, `3 минуты`, `1 день`.

Окно всегда `[сейчас - depth ; сейчас]`.

## Примеры

Проверка:

```bash
curl -s http://127.0.0.1:8080/healthz
```

Расчёт:

```bash
curl -G 'http://127.0.0.1:8080/api/collect' \
  --data-urlencode 'host=saymon.example.com' \
  --data-urlencode 'token=682a3631-cfb2-49dd-b09f-858280a240dd' \
  --data-urlencode 'object_id=6a61ae4562e391eba8d3edbb' \
  --data-urlencode 'metrics=message.I' \
  --data-urlencode 'metrics=message.Temp' \
  --data-urlencode 'aggregates=avg' \
  --data-urlencode 'aggregates=min' \
  --data-urlencode 'aggregates=max' \
  --data-urlencode 'aggregates=sum' \
  --data-urlencode 'aggregates=dev' \
  --data-urlencode 'depth=12h'
```

Токен заголовком:

```bash
curl -G 'http://127.0.0.1:8080/api/collect' \
  -H 'X-Saymon-Token: 682a3631-cfb2-49dd-b09f-858280a240dd' \
  --data-urlencode 'host=saymon.example.com' \
  --data-urlencode 'object_id=6a61ae4562e391eba8d3edbb' \
  --data-urlencode 'metrics=message.I' \
  --data-urlencode 'aggregates=avg' \
  --data-urlencode 'depth=3m'
```

Через nginx (если настроен как в [nginx.md](nginx.md)):

```bash
curl -G 'https://recollect.example.com/api/collect' \
  -H 'X-Saymon-Token: 682a3631-cfb2-49dd-b09f-858280a240dd' \
  --data-urlencode 'host=saymon.example.com' \
  --data-urlencode 'object_id=6a61ae4562e391eba8d3edbb' \
  --data-urlencode 'metrics=message.I' \
  --data-urlencode 'aggregates=avg,min,max,sum,dev' \
  --data-urlencode 'depth=24 часа'
```

## Ответ

`200` — расчёт выполнен (часть детей может быть в `skipped` или с `error` у конкретного ребёнка).

```json
{
  "host": "saymon.example.com",
  "object_id": "6a61ae4562e391eba8d3edbb",
  "depth": "12h",
  "from": 1758960000000,
  "to": 1759003200000,
  "metrics": ["message.I", "message.Temp"],
  "aggregates": ["avg", "min", "max", "sum", "dev"],
  "results": [
    {
      "id": "6a61aea562e391eba8d3edc3",
      "name": "ИКЗ 2603017001",
      "matched_metrics": ["message.I", "message.Temp"],
      "values": {
        "message.I": {"avg": 1.2, "min": 0.4, "max": 3.1, "sum": 48.0, "dev": 0.7},
        "message.Temp": {"avg": 22.0, "min": 18.1, "max": 27.4, "sum": 880.0, "dev": 2.1}
      }
    }
  ],
  "skipped": [
    {
      "id": "6a61aed362e391eba8d3edd7",
      "name": "ИКЗ 2603017002",
      "reason": "no matching metrics in metrics_cache"
    }
  ]
}
```

Коды ошибок:

| Код | Когда |
| --- | --- |
| `400` | нет обязательного параметра или некорректные агрегаты/глубина |
| `502` | SAYMON не ответил на список детей |
| `504` | истекло время ожидания запроса |

`from` и `to` — миллисекунды Unix, как в SAYMON.
