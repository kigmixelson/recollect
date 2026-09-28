# Использование

Сервис слушает HTTP. После [установки](install.md) проверьте liveness и вызывайте расчёт агрегатов.

## Эндпоинты

| Метод | Путь | Назначение |
| --- | --- | --- |
| `GET` | `/healthz`, `/recollect/healthz` | Проверка, что процесс жив |
| `GET` | `/api/collect`, `/recollect/api/collect` | Расчёт, параметры в query |
| `POST` | `/api/collect`, `/recollect/api/collect` | Расчёт, JSON в теле — для HTTP-проверки SAYMON |

Префикс `/recollect` включён по умолчанию (`HTTP_PREFIX=/recollect`), чтобы встроить сервис в уже существующий nginx-хост: `https://<хост>/recollect/api/collect`.

## Как считается ответ

1. Берётся список детей `GET /node/api/objects/{id}/children`.
2. Запрошенные метрики накладываются на `metrics_cache` каждого ребёнка.
3. Для совпавших метрик запрашивается история на указанную глубину.
4. У каждого ребёнка берётся **последняя точка** в этом окне.
5. По этим свежим значениям считается агрегат (среднее, min, max, …) **сразу по всем детям**.

Если в окне нет ни одной точки, метрика возвращается пустой: `"message.I": {}`.

## Параметры `/api/collect`

`GET` — query string. `POST` — JSON в теле. Заголовки перекрывают токен и хост.

Значения с пробелами в GET кодируйте (`curl -G --data-urlencode`).

| Параметр | Обязательный | Пример | Описание |
| --- | --- | --- | --- |
| `host` | да | `http://pult.dc-en.ru` | Хост SAYMON. Можно с схемой `http://` или `https://`. Без схемы берётся `https` |
| `scheme` | нет | `http` | `http` или `https`, если в `host` нет схемы |
| `token` | да | `682a3631-...` | Ключ доступа SAYMON |
| `object_id` | да | `6a61ae4562e391eba8d3edbb` | ID родительского объекта |
| `metrics` | да | `message.I` | Метрики: повторять параметр или перечислить через запятую |
| `aggregates` | нет | `avg` | По умолчанию `avg`. Можно несколько: min, max, sum, dev |
| `depth` | да | `12h` | Глубина окна от текущего момента |

Синонимы query-параметров:

- хост: `host`, `hostname`, `saymon_host`, заголовок `X-Saymon-Host`
- схема: `scheme`, `proto`, `protocol`
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
| URL | `https://<хост>/recollect/api/collect` |
| HTTP headers | `Content-Type` = `application/json` |
| Request body | JSON ниже |
| HTTP Auth | `Bearer`, токен в поле username |
| Timeout | `120000` (мс), запросы к истории могут быть долгими |
| Send response body | включено |
| Response format | JSON, если есть в списке, иначе Autodetect |

```json
{
  "host": "http://pult.dc-en.ru",
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

История запрашивается без downsample: из окна берётся последняя точка каждого ребёнка, агрегат считается уже по этим значениям.

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
  --data-urlencode 'host=http://pult.dc-en.ru' \
  --data-urlencode 'token=YOUR_TOKEN' \
  --data-urlencode 'object_id=6a61ae4562e391eba8d3edbb' \
  --data-urlencode 'metrics=message.I' \
  --data-urlencode 'aggregates=avg' \
  --data-urlencode 'depth=12h'
```

Без схемы в `host` сервис ходит по **https**. Если SAYMON на http, пишите `host=http://pult.dc-en.ru` или добавьте `--data-urlencode 'scheme=http'`.

Не оставляйте пустой `--data-urlencode` без значения — curl ломает query (`Could not parse the URL`).

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

`200` — расчёт выполнен. `values` — итог по всем детям, `samples` — свежие точки, из которых он собран.

Пример для трёх ИКЗ и `message.I` = 52.625, 42.25, 37:

```json
{
  "host": "https://pult.dc-en.kg",
  "object_id": "6a61ae4562e391eba8d3edbb",
  "depth": "12h",
  "from": 1790573586417,
  "to": 1790616786417,
  "metrics": ["message.I"],
  "aggregates": ["avg"],
  "values": {
    "message.I": {"avg": 43.958333333333336}
  },
  "samples": [
    {"id": "6a61aea562e391eba8d3edc3", "name": "ИКЗ 2603017001", "metric": "message.I", "value": 52.625, "timestamp": 1790616700000},
    {"id": "6a61aed362e391eba8d3edd7", "name": "ИКЗ 2603017002", "metric": "message.I", "value": 42.25, "timestamp": 1790616700000},
    {"id": "6a61af1362e391eba8d3edea", "name": "ИКЗ 2603017003", "metric": "message.I", "value": 37, "timestamp": 1790616700000}
  ]
}
```

Нет точек в окне:

```json
{
  "values": {
    "message.I": {}
  }
}
```

Коды ошибок:

| Код | Когда |
| --- | --- |
| `400` | нет обязательного параметра или некорректные агрегаты/глубина |
| `502` | SAYMON не ответил на список детей |
| `504` | истекло время ожидания запроса |

`from` и `to` — миллисекунды Unix, как в SAYMON.
