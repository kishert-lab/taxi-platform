# GIS Backend Audit

Дата: 2026-09-16. Репозиторий: `taxi-platform`, HEAD `5a4e13f` **плюс текущее рабочее дерево**.

Проверены исходники, конфигурация, Compose, Swagger и документация. GIS pipeline, routing и этот документ уже присутствовали как незакоммиченные файлы до начала аудита; это не подтверждает их развёртывание. Изменён только этот документ. Docker CLI в текущем PowerShell не найден; реальные контейнеры, volumes, production environment и мобильный клиент не проверялись. Секретные `.env` не читались.

**Вывод:** основа self-hosted MVT → origin → proxy, OSRM и Pelias уже реализована в рабочем дереве. Минимальный путь — проверить и довести существующий pipeline до эксплуатации. Новый параллельный GIS stack не нужен. По умолчанию map release и OSRM не настроены; документированный pilot не доказывает готовность production.

## 1. Current architecture

### CURRENT — связи в коде; активация отдельно

```text
Build/update (требует явного запуска на Linux/Docker host)
  Geofabrik OSM PBF + MD5 по HTTPS
    → manage.py → releases/<version>/
       ├─ Tilemaker → map.mbtiles (gzip MVT/PBF)
       ├─ OSRM extract/partition/customize → автомобильный MLD graph
       ├─ Pelias schema + OSM importer → отдельный Elasticsearch index
       └─ local style.json + sprites + Noto glyphs + attribution

Runtime, если настроен и развёрнут
  mobile/web → HTTPS edge → nginx map-proxy → map-origin → MBTiles/assets
  mobile/web → taxi API → maps.Service → Redis CachedService → OSRM client
  passenger estimate → тот же routing.Service → тарифы парков → domain pricing
  search → local_geo_points (PostGIS) → Pelias → optional DaData/Yandex
  reverse → maps.Service → Pelias
  driver location REST → backend → PostgreSQL + Redis realtime → WebSocket

Другие существующие deployment paths
  docker-compose.yml / docker-compose.prod.yml → фиксированный Pelias stack
  docker-compose.deploy.yml → PostgreSQL + Redis + backend, без GIS services
```

В [configs/config.yaml](configs/config.yaml) `maps.public_url`, `maps.data_version`, `routing.osrm_url` пустые. При таких значениях map config возвращает 503, routing — `ErrUnavailable`; passenger estimate сообщает недоступность цены. Pelias URL задан, но URL не доказывает наличие импортированных данных.

[docs/maps-validation.md](docs/maps-validation.md) описывает pilot Monaco от 2026-09-10 на другом хосте: импорт, OSRM, origin/proxy, headless MapLibre. Там прямо указано отсутствие production activation. Это историческое свидетельство из документа, не повторная проверка в этом аудите.

## 2. Existing components

| Компонент | Источник | Что можно переиспользовать |
| --- | --- | --- |
| Pipeline | [manage.py](infrastructure/maps/manage.py) | build, validate, activate, rollback, update, prune, status, prepare-transfer; lock/state |
| Tilemaker | [tilemaker.json](infrastructure/maps/tilemaker.json), [process.lua](infrastructure/maps/process.lua) | Собственная MVT schema, offline generation |
| Origin | [server.py](infrastructure/maps/server.py), [Dockerfile](infrastructure/maps/Dockerfile) | Python WSGI/Gunicorn, SQLite read-only, local assets; 2 workers × 4 threads |
| Proxy | [maps Compose](docker-compose.maps.yml), [nginx template](infrastructure/maps/nginx.conf.template) | Cache, CORS, gzip; не проксирует business API |
| Routing | [routing.go](internal/routing/routing.go), [OSRM client](internal/routing/client/osrm/client.go) | Типизированная граница, валидация, GeoJSON, meters/seconds |
| Route cache | [route_cache.go](internal/redis/route_cache.go) | Redis successful-route cache и latency histogram |
| Geocoding | [service.go](internal/geocoder/service/service.go), [Pelias client](internal/geocoder/client/pelias/client.go) | Local points, Pelias, отключаемые hosted fallbacks |
| Storage | [migration 21](migrations/000021_hybrid_geocoder.up.sql), Compose | PostGIS — бизнес-геоданные/адреса; Elasticsearch — Pelias; Redis — runtime |
| Realtime | [ws handler](internal/transport/http/handler/ws_handler.go), [gateway](internal/redis/realtime_gateway.go) | JWT, Redis Pub/Sub, REST resync |
| Update schedule | [timer](infrastructure/maps/taxi-maps-update.timer), [service](infrastructure/maps/taxi-maps-update.service) | Шаблон monthly build + activate + prune; установка не подтверждена |

### Порты и сети

Это **объявленные настройки**, не сканирование хоста. `—` означает отсутствие host port mapping, а не недоступность из других контейнеров.

| Compose / сервис | Host → container по умолчанию | Сеть / данные |
| --- | --- | --- |
| dev `postgres` | все интерфейсы `5432 → 5432` | `network-taxi-api`; `taxi_pg_data` |
| dev `redis` | все интерфейсы `6379 → 6379` | та же сеть; `taxi_redis_data` |
| dev `pelias-elasticsearch` | все интерфейсы `9200 → 9200` | та же сеть; `taxi_pelias_es_data` |
| dev `pelias-libpostal` | все интерфейсы `4400 → 4400` | та же сеть |
| dev `pelias` | все интерфейсы `4000 → 4000` | та же сеть; config bind mount |
| dev `backend` | все интерфейсы `8080 → 8080` | та же сеть |
| prod PostgreSQL/Redis/Elasticsearch/libpostal/Pelias | — | `taxi_internal`, `internal: true`; named volumes |
| prod `backend` | `127.0.0.1:8080 → 8080`, env override | `taxi_internal` + `taxi_public` |
| prod `web` | `0.0.0.0:80 → 80`, env override | обе сети; конфигурация внутри внешнего web image здесь не проверена |
| deploy PostgreSQL/Redis | — | `taxi-platform` bridge; named volumes |
| deploy `backend` | все интерфейсы `${HTTP_PORT:-8080} → 8080` | `taxi-platform` |
| maps `map-proxy` | `127.0.0.1:${MAP_HTTP_PORT:-8090} → 8080` | external `${MAP_NETWORK:-taxi-maps}`; `map_proxy_cache` |
| maps `map-origin` | —; 8000 | та же сеть; `${MAP_DATA_ROOT:-/srv/taxi-maps}:/data:ro` |
| generated `maps-<version>-osrm` | —; 5000 | external `taxi-maps`; release directory `:/data:ro` |
| generated `maps-<version>-es` | —; HTTP 9200 | та же сеть; release `elasticsearch/` bind mount |
| generated `maps-<version>-pelias` | —; 4000 | та же сеть; release `pelias.json` |
| generated `maps-<version>-libpostal` | —; 4400 | та же сеть |

Основания: [dev](docker-compose.yml), [prod](docker-compose.prod.yml), [deploy](docker-compose.deploy.yml), [maps](docker-compose.maps.yml), `Pipeline.generate_compose`. Имена named volumes фактически получают Compose project prefix. Только `taxi_internal` явно объявлена Docker internal network. Pipeline создаёт обычный bridge `taxi-maps`; отсутствие host ports и полная изоляция сети — разные свойства.

**Исправление прежнего черновика:** standalone OSRM service и `/srv/osrm/data` в текущих Compose отсутствуют. OSRM описан генератором release Compose. Дублируются варианты Pelias deployment, а не два явно объявленных OSRM service.

## 3. Current data flow

1. [config.example.json](infrastructure/maps/config.example.json) задаёт `https://download.geofabrik.de/russia-latest.osm.pbf`, MD5 URL, bounds, контрольные точки и `/srv/taxi-maps`. Это пример, не действующая конфигурация хоста. `min_free_bytes: null` блокирует build до задания бюджета диска.
2. `Pipeline.source` получает MD5 и скачивает PBF через curl: HTTPS-only redirects, retry 3, connect timeout 15 s. Manifest сохраняет SHA-256 скачанного PBF. MD5 здесь проверка файла, не подпись поставщика.
3. `tiles`, `graph`, `geocode` используют один `region.osm.pbf`. Tilemaker создаёт MBTiles; OSRM — graph по `/opt/car.lua`; Pelias — индекс `pelias` в отдельном release Elasticsearch. PostgreSQL не является источником base-map tiles.
4. `assets` создаёт дневной стиль и sprites; Noto glyph archive скачивается при подготовке с закреплённым SHA-256. Runtime URL стиля локальные. [images.lock.json](infrastructure/maps/images.lock.json) закрепляет GIS images по digest.
5. Build проверяет SQLite integrity, непустой tileset, слои одного контрольного tile, assets, OSRM и Pelias probes. Успешный build пишет `validated.json`; без него origin не отдаёт release. Smoke checks не доказывают полноту региона.
6. `activate` повторно проверяет release, генерирует backend Compose override с URL/version и maps network, пересоздаёт backend, сверяет `data_version` в `/public/map/config`. Пишет `active.json`; при ошибке пытается вернуть прежний override.
7. `update` выполняет build → activate → prune. `prune` сохраняет active/previous и минимум два последних validated release. Старые runtime containers до удаления могут оставаться запущенными.

Persistent layout: `/srv/taxi-maps/releases/<version>/` содержит PBF, `map.mbtiles`, `region.osrm*`, `elasticsearch/`, `pelias-work/`, `public/`, configs, log/state/validation files. В корне — lock, active/activation state, backend overrides. Промежуточные файлы тоже занимают диск. Pilot path `/media/d/taxi-maps-pilot/releases/pilot-20260909b` не является подтверждённым production volume.

## 4. Existing API endpoints

Префикс всех API путей ниже — `/api/v1`. Доступ установлен по **полной цепочке middleware**: [main.go](cmd/api/main.go), [authentication.go](internal/middleware/authentication.go), handlers. Отсутствие auth в конкретном handler не делает endpoint публичным.

| Метод и путь | Доступ | Назначение |
| --- | --- | --- |
| `GET /public/map/config` | public | style URLs, bounds, zoom, version, attribution, availability |
| `POST /map/routes` | общий user access JWT | Две точки → road geometry/distance/duration |
| `POST /passenger/map/routes` | passenger access JWT | Тот же routing response |
| `GET /geocoder/reverse` | общий user access JWT | `lat`, `lon` → адрес отдельно от исходной точки |
| `GET /passenger/address/reverse` | passenger access JWT | Тот же reverse contract |
| `GET /geocoder/search` | общий user access JWT | Гибридный поиск, `q`, city/focus/limit |
| `GET /passenger/address/search` | passenger access JWT | Поиск через passenger service |
| `POST /geocoder/points/confirm` | общий user access JWT | Подтверждение локальной точки |
| `GET/POST /admin/geocoder/local-points` | JWT + admin role | Локальные точки |
| `POST /admin/geocoder/local-points/{id}/approve`, `/reject` | JWT + admin role | Модерация |
| `GET /admin/geocoder/export/pelias-csv` | JWT + admin role | Экспорт, не автоматический release import |
| `POST /passenger/orders/estimate` | passenger access JWT | Road metrics + тарифы парков |
| `POST /taxi-park/orders/estimate` | JWT + taxi_park/dispatcher role | Выбранный тариф; сервис проверяет доступ к парку |
| `GET /driver/orders/{id}/route` | user JWT + driver role / order access | Сохранённый GPS-трек, не OSRM route calculation |
| `POST /driver/orders/{id}/route/batch` | те же проверки | Запись точек поездки |
| `POST /driver/location`, `/driver/location/batch` | user JWT + driver role | Обновление положения |
| `GET /ws` | JWT в handshake (header/query fallback) | Существующий realtime protocol |

Origin предоставляет `GET/HEAD /releases/{version}/tiles/{z}/{x}/{y}.pbf` и assets под тем же release prefix: `style.json`, `sprite.json/png`, `sprite@2x.json/png`, `fonts/{fontstack}/{range}.pbf`, license files. Bare `/tiles/...` не зарегистрирован. Assets application-facing, без JWT; proxy разрешает также OPTIONS. Внутренние OSRM/Pelias URL клиентам не выдаются.

Swagger: [swagger.yaml](docs/swagger.yaml), [swagger.json](docs/swagger.json), [docs.go](docs/docs.go); UI `/swagger/*any`. Map/reverse/search/estimate/driver-route пути присутствуют. Прежнее утверждение о public `/map/routes` и `/geocoder/reverse` неверно: они защищены глобальным JWT middleware. Существующий `TestMapRoutesKeepPassengerAndUserTokensSeparate` описывает разграничение token types; в этом аудите он только прочитан.

## 5. Existing map format

**Base map — MVT/PBF в MBTiles; raster basemap pipeline не найден.** PNG sprites — значки, не raster tiles. Origin переводит XYZ `y` в TMS tile row SQLite. Geometry не содержит цветов; отрисовку выполняет клиент.

| Layer | Содержание по process.lua | Zoom по schema / фильтрам |
| --- | --- | --- |
| `road` | highway, class, русское или исходное name | 5–14; кроме motorway/trunk/primary — от 11 |
| `building` | Замкнутые здания | 13–14 |
| `housenumber` | Номера узлов и centroid замкнутых объектов | 14 |
| `place` | Название и class | schema от 3; города от 5, прочие от 10 |
| `poi` | amenity/shop/tourism, только name | 13–14 |
| `water` | Замкнутые natural=water / riverbank | 6–14 |
| `landuse` | landuse и leisure=park | 8–14 |

Origin принимает zoom 0–14; непустой tile на каждом zoom не гарантирован. Style source maxzoom=14; API maxzoom=22 — диапазон отображения с overzoom, не tiles z15–22. Номера домов в стиле включаются с zoom16. Source maxzoom допускает такое переиспользование tiles ([MapLibre specification](https://maplibre.org/maplibre-style-spec/sources/)).

Нет явных слоёв boundaries, railway, airports, entrances; parking может попасть в generic POI без класса. Все amenity/shop/tourism включаются без taxi-specific allowlist. Нет отдельной обработки coastline/open waterway в Lua; полноту воды, сложных relations и транспортных объектов надо проверять на региональном наборе. Полная адресная база в tiles не передаётся.

### Caching и style

- Существующие tiles/assets: SHA-256 ETag, точное If-None-Match → 304, `Cache-Control: public,max-age=31536000,immutable`. Last-Modified не реализован. Missing tile → 204 с TTL 1 day; отсутствующий release/asset → 404 no-store; I/O error → 503 no-store.
- MVT заранее gzip-сжат; origin выставляет Content-Encoding. nginx gzip настроен для JSON/glyph/MVT MIME types. Brotli, HTTP/2 и HTTP/3 в этих configs не включены; HTTPS sample содержит только `listen 443 ssl`.
- nginx: cache 2 GiB, metadata zone 20 MiB, inactive 30 days, URI key, lock и revalidation. Query не участвует в ключе — допустимо, пока URI полностью определяет asset. Cache lock уже предотвращает одновременное заполнение одного ключа ([nginx documentation](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_cache_lock)).
- Есть только `day`/`style.json`. Assets отделены от geometry, но **независимого style version lifecycle и night theme нет**: весь validated release immutable, builder связывает стадии одним version.
- Style source не получает региональные bounds. Мобильный cache, eviction и Android 9 performance находятся вне backend repo и здесь не доказаны.

## 6. Existing routing implementation

[routing.Service](internal/routing/routing.go): `Route(context.Context, origin, destination) (Route, error)`. [routes.go](cmd/api/routes.go) связывает application services с OSRM client и Redis decorator. Routing остаётся на сервере.

OSRM request: `/route/v1/driving/{lon},{lat};{lon},{lat}?overview=full&geometries=geojson&steps=false&radiuses=...`. MLD graph создаётся из PBF. Distance/duration вычисляет OSRM; адаптер проверяет status/code, наличие metrics, finite/nonnegative значения, LineString, waypoints, координаты, coverage bounds (включая antimeridian), snap radius. Response limit 2 MiB; timeout в wiring 3 s. В client одна HTTP попытка, bounded retry/backoff нет. Источник live traffic в pipeline не настроен.

Ответ: целые `distance_meters`, `duration_seconds`, GeoJSON `geometry`, `original_points`, `snapped_points`, `source`, `data_version`, `calculated_at`. Последнее — timestamp, не длительность вычисления. Geometry использует `[longitude,latitude]`, input objects — `latitude`/`longitude`. Route API принимает ровно две точки. Encoded polyline сейчас не реализован.

Redis сохраняет successful responses. Namespace включает URL, version, driving/full/geojson, snap radius, bounds; ключ также включает точки. Activation override задаёт TTL 5 min. Cache errors логируются, запрос проходит в OSRM. Distributed coalescing одинаковых cache misses не найден. При ошибке регистрации metrics wiring оставляет прямой OSRM client без cache.

### Pricing и другие distance/duration

- Passenger estimate и создание заказа вызывают `buildTaxiParkEstimate`: OSRM → доступные тарифы парков → [domain.CalculateTripPrice](internal/domain/trip_pricing.go) → average/min/max price и snapshot metrics/rates/version. Ошибка routing даёт unavailable price, а не маршрут по прямой.
- [taxipark/routing.go](internal/taxipark/routing.go) применяет выбранный park tariff. Domain поддерживает fixed/distance/time/distance_time, minimum price, округление денег и начатых минут. Pricing остаётся в taxi-platform. Route duration не равна ETA подачи; отдельная road-based pickup ETA здесь не доказана.
- Сохранился `buildPassengerEstimate → estimateOrder → repository.EstimateRoute`: PostGIS `ST_Distance` между двумя geography points и оценка времени 30 km/h. По поиску ссылок helper не вызывается текущим production path. Это остаточный код, не действующий OSRM fallback.
- [driver_mobile_repository.go](internal/repository/driver_mobile_repository.go) считает фактический пробег по последовательным GPS-точкам `order_route_points` и время поездки. История трека отличается от планирования маршрута.

## 7. Existing geocoding implementation

[Search service](internal/geocoder/service/service.go): нормализация → city/focus context → trusted local points в PostGIS → Pelias → DaData → Yandex. При confidence ≥ threshold (default 0.75) Pelias сразу возвращается; иначе проверяются fallback providers, затем допускается исходный слабый Pelias result. При city prefix возможна повторная попытка без него.

Yandex/DaData defaults отключены, но включаются env. PostgreSQL cache внешних ответов читается до проверки enabled provider: отключение прекращает новые вызовы, но не убирает ранее cached результаты. `PeliasCacheTTLDays` есть в config, однако успешный Pelias Search возвращается напрямую, не через cache writer внешних providers. Нельзя считать все Pelias ответы кешируемыми на сервере.

Reverse использует отдельный Pelias client через maps.Service без local-points/fallback chain. Возвращает неизменный `point`, `found`, `address`, отдельный `address_location`, `source`; pickup автоматически не заменяется. Client проверяет HTTP/JSON/coordinates, но пропускает некорректные features без диагностического события; geometry type, обязательность label и confidence range полностью не проверяются. Retry/backoff нет. Ошибка Pelias Search может стать пустым результатом: итоговый unavailable flag учитывает ошибки DaData/Yandex, но не Pelias.

### Pelias imports

- Фиксированный [pelias.json](configs/pelias.json) задаёт ES host, index, libpostal и focus на Пермь. Dev/prod Compose **не содержат OSM importer job**; происхождение содержимого `taxi_pelias_es_data` по ним не устанавливается.
- Release pipeline запускает `pelias/schema` и `pelias/openstreetmap` над тем же PBF, пишет LevelDB в `pelias-work`, ES в release directory, проверяет count/search/reverse. Используется pinned `pelias/elasticsearch`; фиксированный stack использует обычный Elastic 7.17.23. Pilot report отмечает необходимость ICU; старый stack требует отдельной проверки совместимости.
- `imports.adminLookup.enabled=false`; WOF/OpenAddresses/interpolation imports не описаны. Nonempty probe не доказывает полноту домов/подъездов/муниципалитетов. Admin CSV export существует, его автоматическая загрузка в release Pelias не найдена. Local-first поиск сохраняет эти точки доступными через taxi API.

### Realtime contract

Событие **`driver.location_updated`**, envelope `event`, `request_id`, `occurred_at`, `payload` ([message.go](internal/ws/message.go)). Passenger payload: `order_id`, `driver_id`, `status`, `location:{latitude,longitude}`, optional heading/speed/accuracy, `recorded_at` ([passenger_payloads.go](internal/ws/passenger_payloads.go)). Пример `type: driver.location` из задачи не является текущим контрактом.

Driver service определяет текущий заказ и отправляет событие его пассажиру через Redis gateway. Coordinates сохраняются отдельно от tiles; Redis используется для Pub/Sub, presence и throttling. После reconnect сервер отправляет `sync.required`; клиент восстанавливает current order через REST. Pub/Sub не обеспечивает долговременный replay. Base tiles при движении не пересобираются.

## 8. Problems/gaps

| Приоритет | Пробел / основание | Следующее действие |
| --- | --- | --- |
| High | Production release/domain/image/data не подтверждены; defaults пусты | Read-only inventory реального deployment, региональный smoke test |
| High | Fixed Pelias и release Pelias — разные datasets/deployment paths | Выбрать authoritative путь после проверки потребителей; сохранить rollback |
| High | `backend_override` не передаёт bounds из manifest; backend default — Россия, pilot может быть другим | Передавать и сверять реальные bounds/zoom/version при активации |
| High | Activation проверяет backend config version; availability вычисляется из settings, не live health | Проверять полный HTTPS asset + routing/geocoding путь, не трактовать flags как SLA |
| High | Dev открывает DB/Redis/ES/libpostal/Pelias на всех интерфейсах; ES security выключена | Не использовать dev topology для production; проверить firewall/network membership |
| Medium | Hosted providers включаются env, activation их не отключает | Enforce self-hosted policy в deployment и runtime egress |
| Medium | Malformed Pelias features пропускаются, outage может стать empty result | Разделить no-results/invalid/unavailable; bounded retries в пределах deadline |
| Medium | Night theme и independent style revisions отсутствуют | Отдельное versioning assets с повторным использованием tiles |
| Medium | Нет tile byte/feature budgets и полного coverage QA | Измерить p95/max payload, dense-city rendering, transport coverage |
| Medium | Нет специальных proxy/geocoder метрик; routing смешивает cache/upstream | Панели и метрики из раздела 12 |
| Medium | Full rebuild и retained ES/OSRM runtime увеличивают ресурсы | Измерить simultaneous peak, определить остановку старых services и retention |
| Medium | Activate force-recreate одного backend; rollback не проверен здесь | Staging interruption/resync, first activation failure, rollback |
| Low | Остаточный straight-line pricing helper | Отдельно удалить после проверки ссылок, не включать как road fallback |

Origin читает asset/tile и пересчитывает hash на origin request, включая HEAD; proxy cache существенно важен. Generic public-assets path обслуживает любой файл в `public/`: там должны быть только клиентские assets. GIS admin HTTP endpoint у origin нет. Backend `/metrics` зарегистрирован без JWT; scrape access зависит от deployment proxy.

## 9. Proposed vector tile architecture

### TARGET — дополнить существующие компоненты

```text
OSM extract + provenance/checksum
  → existing release builder
     ├─ Tilemaker → immutable data-version MBTiles/MVT
     ├─ OSRM MLD graph (same PBF)
     └─ Pelias index (same PBF)

Versioned day/night style + local glyphs/sprites
  → existing read-only origin → nginx cache → HTTPS edge
  → bounded device cache → decode/style/GPU rendering

taxi API → routing.Service → Redis cache → internal OSRM
taxi API → local PostGIS points / internal Pelias
driver → existing REST / Redis / WebSocket → passenger marker
```

Повторно использовать origin/proxy/Tilemaker; business PostGIS не превращать во второй tile stack. Сохранить immutable URLs и существующий public contract. OSM source/fonts/images скачиваются при подготовке, а не по запросу пассажира.

После аудита: independent style revision, day/night на одном tileset; документированная schema; точечные airport/station/parking/entrance classes при подтверждённом покрытии. Ограничить POI, детализацию и свойства по zoom на основе измерений. Не передавать весь город и realtime taxi data внутри MVT.

Сначала использовать имеющиеся gzip/ETag/cache, проверить 200/204/304, CORS, encoding и client cache. HTTP/2 включать на реальном TLS edge после проверки image/version; HTTP/3/Brotli не обязательны для первого запуска. Разделение source/style соответствует [MapLibre specification](https://maplibre.org/maplibre-style-spec/sources/).

## 10. Proposed routing boundary

Сохранить `routing.Service` и OSRM adapter. Handler валидирует transport; application service выполняет use case; domain pricing применяет tariff; Redis/HTTP clients остаются infrastructure. Routing возвращает географические характеристики, не стоимость или бизнес-ETA.

Будущее compact encoding: additive/versioned поле либо согласованная опция `geometry_format` с точностью (например polyline6), координатным порядком и единицами. Сохранить GeoJSON compatibility. OSRM поддерживает polyline/polyline6/geojson, но это возможность upstream, не текущий passenger API ([OSRM API](https://project-osrm.org/docs/v5.24.0/api/)). Нельзя молча заменить объект `geometry` строкой.

Manifest связывает PBF/tiles/graph/index; style revision меняется отдельно при совместимой schema. Cache namespace сохраняет data/algorithm options. Проверка `maps.data_version == routing.data_version` уже есть; дополнительно нужны actual dataset/bounds/upstream health, не только совпадение строк.

## 11. Migration plan

1. **Этот этап:** сохранить audit и остановиться. Не запускать build/activate/update, не менять API, Compose, routing engine или GIS data.
2. Далее выполнить read-only inventory: effective Compose без вывода secrets, networks/ports, image digests, mounts, active release/settings. Сверить рабочее дерево с deployed commit/image. Не считать fixed Pelias пустым без проверки.
3. Принять release-managed pipeline как целевой путь, подготовить миграцию geocoding consumers и local points. Выводить fixed Pelias из эксплуатации после проверенного переключения/rollback.
4. Исправить activation bounds/policy, error semantics, limits; настроить реальный origin image, HTTPS/CORS, monitoring. Проверить региональные адреса/маршруты/tiles. Не включать monthly `update` до ручного цикла: он автоматически активирует release.
5. Собрать pilot фактической зоны обслуживания с measured disk budget. Проверить invalid/out-of-coverage/unreachable routes, snap radius, geocoding errors, pricing/road metrics, сохранение pickup. Не переносить Monaco результат на Россию.
6. Проверить Expo/React Native на Android 9: tiles/style/fonts, кириллицу, cold/warm cache, память, pan/zoom/overzoom, route overlay, driver marker, reconnect. Native renderer/package/build — отдельная mobile задача.
7. Проверить staging activation/rollback, включая смену региона и отказ upstream. Затем переключить backend/release по принятой процедуре и наблюдать нагрузку. Ночной стиль и compact route contract — отдельные совместимые изменения.

Критерии готовности: подтверждённый согласованный dataset, только self-hosted runtime requests, совместимость клиентов/rollback, закрытые internal services, измеренные latency/payload/resource budgets. Миграция в этом аудите не выполнялась.

## 12. Resource implications

Production CPU/RAM/disk не измерены. Pilot report приводит 143 267 324 bytes Monaco artifacts; это не production sizing. Example `build_memory=4g`, `build_cpus=2`, `es_heap=1g` — параметры шаблона, не доказанная достаточность для России. Build limits также применены к OSRM runtime. ES heap не ограничивает весь RSS.

| Ресурс | Основные затраты | Измерения |
| --- | --- | --- |
| CPU | Tilemaker, OSRM preprocessing, Pelias import; runtime route/search/hash/gzip | Peak, stage duration, нагрузочный профиль |
| RAM | Build, OSRM graph, ES heap + filesystem cache, workers | RSS каждого компонента и active/previous/new одновременно |
| Disk | PBF/MBTiles/graph/index/scratch/fonts/retained releases | Active + previous + новый build + temporary space + запас; nginx cache 2 GiB отдельно |
| Network | Build downloads, tile egress, driver updates | Байты на сессию, cold/warm cache, update downloads |
| Mobile | Decode, labels, GPU, tile cache | Frame time, memory, disk и payload budgets Android 9 |

Уже есть `taxi_routing_duration_seconds{status="cache_hit|success|error"}` в Redis decorator и backend `/metrics`. Histogram позволяет вывести rate/errors/quantiles, но измеряет всю операцию с Redis; отдельной OSRM calculation duration нет. Pipeline пишет stage durations и artifact bytes. Специализированные GIS dashboards/exporters в проверенных файлах не найдены.

Необходимое дополнение:

- Tiles: request rate, p50/p95/p99, bytes, errors, cache HIT/MISS/BYPASS. Текущий nginx template не добавляет cache status/upstream latency в отдельный structured log и не экспортирует GIS metrics.
- Routing: end-to-end/upstream latency отдельно, timeout/error reasons, cache ratio, request rate. Не использовать координаты/order_id как Prometheus labels.
- Geocoding: local/Pelias/cache hits, latency, errors отдельно от no-results, fallback calls (при strict self-hosted должны отсутствовать).
- Resources: CPU/RAM, disk free/IO, network, build duration, dataset age, retained releases; alerts на нехватку места, outage и задержку обновления.
- Логи: request_id/user_id/operation в application path; version/stage в pipeline; correlation с proxy/origin. Origin error log содержит operation/error, но не request_id.

## 13. Risks

1. **Deployment неизвестен:** незакоммиченная реализация может отличаться от production image; наличие кода не доказывает развёртывание.
2. **Coverage/качество:** OSM неполон для подъездов/адресов, один probe не гарантирует регион. Russia bounds пересекают antimeridian; renderer требует проверки.
3. **Snapshot mismatch:** activation не переносит bounds; fixed Pelias или stale env могут дать разные map/routing/address datasets.
4. **Availability:** force-recreate backend обрывает соединения; rollback может не сработать при нехватке ресурсов/неверном image. Redis Pub/Sub требует REST resync.
5. **Release lifecycle:** prune может удалить assets старого client config; retention должен учитывать срок поддержки клиентов. ES остаётся writable — immutable publication не означает read-only весь release directory.
6. **Capacity:** full rebuild и несколько ES/OSRM sets могут исчерпать RAM/disk при небольшом размере итоговых tiles. Monthly update автоматически переключает backend.
7. **Security:** dev ports, membership maps network и `/metrics` требуют проверки deployment; CORS не заменяет auth/isolation.
8. **Portability:** pipeline использует Linux `fcntl`, `os.chown`, systemd и доступ к container IP. Windows workspace не подтверждает его запуск на Windows; unit содержит конкретный `/media/d/taxi-platform` path.
9. **Provider independence:** hosted fallback env или внешние style/font URLs нарушают runtime цель. `audit_style` помогает, но нужен egress test клиента.
10. **Mobile:** headless Chrome не заменяет native Android 9; большие GeoJSON routes/dense tiles способны перегрузить слабое устройство.

### Результат проверки

Выполнена статическая сверка pipeline, Compose, defaults, routing/pricing/geocoding call paths, JWT middleware, WebSocket и Swagger. Исправлены утверждения прежнего документа о public route/reverse и standalone OSRM. Исходники и deployment не менялись; тесты и GIS jobs в этом documentation-only этапе не запускались. Аудит завершён; дальнейшая реализация относится к отдельному этапу.
