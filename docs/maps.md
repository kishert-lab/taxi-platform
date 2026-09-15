# Собственная карта taxi-platform

Реализация находится в Go-сервисах `internal/maps`, `internal/routing`, OSRM-клиенте, Redis-кеше и `infrastructure/maps`. Клиентские приложения в этом репозитории не изменялись.

## Что изменено

* Подложка: Tilemaker → MBTiles → отдельный WSGI origin под Gunicorn → кеширующий nginx → существующий HTTPS reverse proxy. Go API не обслуживает тайлы.
* Геокодирование: существующий Pelias; добавлен импорт OSM в изолированный Elasticsearch с ICU, обратное геокодирование. Существующие поиск, локальные подтверждённые точки и настройки внешних fallback-провайдеров сохранены.
* Маршруты: OSRM MLD, автомобильный профиль; метры, секунды, GeoJSON, исходные и привязанные точки, версия графа. Контекст, таймаут, лимит ответа, проверка геометрии/waypoints, радиус привязки и границы покрытия. Redis-ключ включает порядок точек, профиль, версию, URL и параметры.
* Пассажирская оценка сохраняет выбор и усреднение доступных тарифов парков. Диспетчерские обычные и запланированные заказы используют тот же доменный расчёт тарифа по дорожным метрикам до начала транзакции записи заказа. Добавлена предварительная диспетчерская оценка.
* `pricing_snapshot` содержит дорожные метрики, версию, ставки, округление и оценку. Завершённые заказы не пересчитываются. GPS-трек и фактические показатели не заменяются прогнозом.
* Новые HTTP-контракты включены в генерируемые `docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go`.

Tilemaker выбран для отдельной пакетной подготовки из PBF без второй географической БД для подложки. Собственная небольшая схема содержит дороги, здания, номера домов, места, POI, водоёмы и землепользование. Стиль, значки и кириллические glyphs размещаются вместе с версией тайлов. Образы подготовки закреплены digest в `images.lock.json`, архив Noto Sans v2.0 проверяется SHA-256. Это базовый дневной стиль; глобальный океанический полигон, рельеф и спутниковая подложка в него не входят.

Первичная проверка существующего сервера обнаружила пустой индекс Pelias (0 документов), работающий API без запущенного OSRM и незавершённые изменения пассажирского расчёта. Они учтены при интеграции.

## Подготовка сервера

Нужны Linux, Docker Engine + Compose с `up --wait`, Python ≥3.10, curl, права создавать контейнеры и назначать владельца каталогов данных. Сборщик использует отдельную Docker-сеть. PostgreSQL, Redis и действующий Elasticsearch не очищаются и не используются для импорта карты.

1. Скопировать `infrastructure/maps/config.example.json` в `/etc/taxi-maps.json`.
2. Указать реальный HTTPS `public_url`, территорию, PBF и checksum URL, контрольные маршрут/адрес/тайл, bounds, ресурсы сборки и Compose-проект бэкенда. Порядок координат в контрольных запросах конфигурации — **longitude,latitude**. Bounds — west,south,east,north; west > east обозначает пересечение 180-го меридиана.
3. `min_free_bytes` намеренно не имеет произвольного значения: задать по пробной сборке и запасу на одновременно существующие активный, предыдущий и новый наборы. `build_memory`, `build_cpus`, `es_heap` — ограничения запуска, не оценка требований России. Проверить также свободное место Docker storage и настройки Elasticsearch, включая `vm.max_map_count`.
4. Задать `backend_compose_files`, `backend_env_file`, `backend_project`, `backend_service` в соответствии с текущим развёртыванием. Проект должен совпадать с существующим Compose project label. Для добавления нового графа backend подключается к дополнительной внешней сети.
5. Для промышленной публикации указать immutable `IMAGE_TAG` API с этими изменениями в существующей конфигурации registry. Команда активации использует `--no-build --no-deps`; она не запускает миграции и не собирает приложение на production.

Собрать origin на build-сервере и опубликовать в вашем registry:

```sh
docker build -f infrastructure/maps/Dockerfile -t REGISTRY/taxi-platform/map-origin:VERSION infrastructure/maps
docker push REGISTRY/taxi-platform/map-origin:VERSION
```

Указать в deployment env:

```dotenv
MAP_ORIGIN_IMAGE=REGISTRY/taxi-platform/map-origin:VERSION
MAP_DATA_ROOT=/srv/taxi-maps
MAP_NETWORK=taxi-maps
MAP_CORS_ORIGIN=https://dispatcher.YOUR-DOMAIN
MAP_HTTP_PORT=8090
```

`VERSION` должен быть неизменяемым тегом или digest. Подключить `docker-compose.maps.yml` отдельным Compose-проектом:

```sh
docker network create taxi-maps # только при первом создании, если сети ещё нет
docker compose --env-file /PATH/maps.env -p taxi-map-assets -f docker-compose.maps.yml up -d
```

Origin не публикует порт; nginx слушает только loopback хоста. Установить сертификат для реального публичного домена и подключить конфигурацию по образцу `infrastructure/maps/https.conf.example` к существующему TLS proxy. При внешнем proxy на другой машине изменить привязку порта с учётом вашей внутренней сети. Мобильные клиенты получают такой же HTTPS URL, что и веб.

## Сборка, проверка, переключение

```sh
python3 infrastructure/maps/manage.py build --config /etc/taxi-maps.json --version russia-202609
python3 infrastructure/maps/manage.py validate --config /etc/taxi-maps.json --version russia-202609
python3 infrastructure/maps/manage.py activate --config /etc/taxi-maps.json --version russia-202609
python3 infrastructure/maps/manage.py status --config /etc/taxi-maps.json
```

`build` скачивает PBF с проверкой MD5 поставщика по HTTPS, сохраняет SHA-256, создаёт тайлы и MLD-граф, поднимает отдельный Elasticsearch и импортирует Pelias, копирует локальные шрифты и создаёт стиль/спрайты. Версия публикуется только после проверки MBTiles, контрольного тайла, дорожного маршрута, ненулевого индекса, поиска и reverse. Пустой индекс считается ошибкой даже при exit code 0 импортера.

В `root/releases/VERSION` находятся отдельные данные, `compose.json`, `pelias.json`, `build.log`, `state.json`, `validated.json`. `validated.json` содержит версии образов, время, исходник и размер артефактов. Статусы и длительности этапов видны в `state.json`. `root/update.lock` предотвращает одновременные запуски на одном управляющем сервере.

Подготовка новой версии не изменяет активную. Возобновление:

```sh
python3 infrastructure/maps/manage.py build --config /etc/taxi-maps.json --version russia-202609
# Повтор конкретного этапа и последующих только для ещё не опубликованного набора:
python3 infrastructure/maps/manage.py build --config /etc/taxi-maps.json --version russia-202609 --from-stage pelias
```

Изменение конфигурации/образов/схемы при возобновлении требует новой версии. Уже проверенные наборы нельзя пересобрать с тем же именем.

`activate` повторно проверяет сервисы, генерирует `root/backend-VERSION.json` с согласованными OSRM/Pelias/map URL и версиями, пересоздаёт только backend и проверяет опубликованную версию через его API. `active.json` записывается после успешной проверки. При ошибке восстанавливается предыдущая конфигурация. Перезапуск единственного API-контейнера может кратко прервать HTTP/WS; клиенты должны восстановить существующее соединение с токеном. Обновление без перезапуска потребовало бы отдельного балансировщика нескольких API-реплик.

При дальнейших штатных deployment-командах **всегда добавлять активный `backend-VERSION.json` последним Compose-файлом**; иначе базовые переменные могут вернуть старый провайдер.

```sh
python3 infrastructure/maps/manage.py rollback --config /etc/taxi-maps.json
# Либо выбрать конкретную сохранённую версию:
python3 infrastructure/maps/manage.py rollback --config /etc/taxi-maps.json --version russia-202608
python3 infrastructure/maps/manage.py prune --config /etc/taxi-maps.json
```

Сохраняются активный и предыдущий релизы и минимум два последних проверенных набора, согласно `retain_releases`. Старые активные клиентские сессии могут запрашивать предыдущий стиль/тайлы. После удаления очень старого набора клиент должен повторно получить конфигурацию. Ошибочные незавершённые наборы автоматически не удаляются: это сохраняет возможность диагностики и повторного запуска.

## Ежемесячное обновление

```sh
sudo install -m 644 infrastructure/maps/taxi-maps-update.service /etc/systemd/system/
sudo install -m 644 infrastructure/maps/taxi-maps-update.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now taxi-maps-update.timer
sudo systemctl start taxi-maps-update.service
journalctl -u taxi-maps-update.service
```

Перед установкой проверить путь репозитория и конфигурации в unit. Расписание задаётся `OnCalendar` timer (по умолчанию первое число месяца, 03:00 UTC, плюс случайная задержка до часа); его можно менять через `systemctl edit taxi-maps-update.timer`. `Persistent=true` обрабатывает пропущенный запуск. `update` использует версию YYYYMM, строит/проверяет, активирует, затем применяет retention. Ошибка любого этапа даёт ненулевой exit code и не считается успешным обновлением.

Сборка может выполняться на отдельном сервере. После проверки остановить **только переносимый релиз на сборщике**:

```sh
python3 infrastructure/maps/manage.py prepare-transfer --config /etc/taxi-maps.json --version russia-202609
```

Перенести каталог релиза целиком, включая Elasticsearch, с сохранением владельцев (1000 для Elasticsearch, 1001 для рабочего каталога импортера). Для согласованной копии Elasticsearch должен быть остановлен. Использовать одинаковый абсолютный root, имена сети и pinned images; перенести образы через registry или `docker save/load`. На целевой машине выполнить `validate`, затем `activate`. `build` и `activate` разделены; таймер может работать на машине, имеющей оба набора ресурсов. Для удалённой сборки планировщик должен организовать перенос до активации.

## Контракты для трёх клиентов

Все пути ниже начинаются с `/api/v1`. Оболочка ответа: `{"data": ..., "meta":{"request_id":"..."}}`.

* Все клиенты: `GET /public/map/config`, без токена. Загрузить `data.style_url` в MapLibre. Сохранить OSM attribution. Источник native tiles имеет zoom 0–14; отображение и overzoom поддержаны до 22, номера домов включаются с zoom 16. Не скачивать весь набор на устройство.
* Пассажир: прежний `GET /passenger/address/search?q=...&lat=...&lon=...`, новый `GET /passenger/address/reverse?lat=...&lon=...`, `POST /passenger/map/routes`. Требуется пассажирский access token.
* Водитель/диспетчер: прежний `GET /geocoder/search?q=...&lat=...&lon=...`, новый `GET /geocoder/reverse?lat=...&lon=...`, `POST /map/routes`. Требуется обычный пользовательский access token.
* Оценка пассажира: прежний `POST /passenger/orders/estimate`. Оценка диспетчера: `POST /taxi-park/orders/estimate`, тело как у создания диспетчерского заказа; проверяются роль, доступ к парку и тарифу. Ответ использует общий `OrderPricingResponse`, деньги — целые копейки.

Пример конфигурации:

```json
{"data":{"style_url":"https://maps.YOUR-DOMAIN/releases/russia-202609/style.json","styles":[{"id":"day","url":"https://maps.YOUR-DOMAIN/releases/russia-202609/style.json"}],"bounds":[19,41,-169,82],"min_zoom":0,"max_zoom":22,"data_version":"russia-202609","updated_at":"2026-09-01T03:00:00Z","attribution":"© OpenStreetMap contributors (ODbL) https://www.openstreetmap.org/copyright","search_available":true,"routing_available":true},"meta":{"request_id":"..."}}
```

Availability означает наличие опубликованного проверенного набора, а не гарантию доступности каждого запроса. При неготовой конфигурации — 503.

Тело запроса маршрута:

```json
{"points":[{"latitude":58.010455,"longitude":56.229443},{"latitude":58.0200,"longitude":56.2500}]}
```

Пример структуры ответа (цифры и геометрия иллюстративны):

```json
{"data":{"distance_meters":2200,"duration_seconds":350,"geometry":{"type":"LineString","coordinates":[[56.229443,58.010455],[56.2500,58.0200]]},"source":"osrm","data_version":"russia-202609","original_points":[{"latitude":58.010455,"longitude":56.229443},{"latitude":58.0200,"longitude":56.2500}],"snapped_points":[{"latitude":58.0104,"longitude":56.2294},{"latitude":58.0200,"longitude":56.2500}],"calculated_at":"2026-09-09T10:00:00Z"},"meta":{"request_id":"..."}}
```

OSRM и GeoJSON используют **longitude,latitude**; именованные API-поля — `latitude`, `longitude`. Поддержаны ровно две точки: в текущей модели заказа промежуточных остановок нет. Неправильный запрос — 400, отсутствие дорожного маршрута/покрытия/допустимой привязки — 422, отказ либо некорректный ответ OSRM — 503. Прямая дистанция не выдаётся за дорожную.

Обратное геокодирование без результата возвращает `{"point":{"latitude":58,"longitude":56},"found":false,"address":"","source":"pelias"}` внутри `data`. Разрешить выбор точки вручную. При найденном адресе `address_location` — отдельная координата дома; `point` не меняется. Не заменять точку у подъезда `address_location` или `snapped_points`.

Для маршрута поездки передать точки доступного заказа. Для маршрута подачи передать позицию назначенного водителя из **существующего авторизованного WebSocket-события** и исходную точку посадки. Новый API маршрута принимает только координаты и не ищет заказы или водителей по ID. Доступ к общему стилю и расчёту геометрии не открывает чужие координаты. Серверные проверки существующих order/WS API продолжают определять, кому доступна позиция водителя и заказ. Не создавать второй realtime-канал.

Расстояние и время — прогноз по автомобильному профилю без живых пробок. Данные OSM о домах, адресах и входах неполны. Импорт OSM без Who's On First использует имеющиеся OSM адресные теги; полнота административной иерархии не гарантируется. Город/фокус существующего геокодера сохраняет значение для поиска, но качество по городам России нужно проверять на целевом наборе.

## Проверки и эксплуатация

```sh
go test ./...
go test -race ./internal/routing/... ./internal/redis ./internal/maps ./internal/passenger ./internal/taxipark
go build -o /tmp/taxi-api-map-check ./cmd/api
(cd infrastructure/maps && python3 -m unittest -v test_maps)
make swagger
```

Контроль опубликованного HTTPS origin:

```sh
curl -I https://maps.YOUR-DOMAIN/releases/russia-202609/style.json
curl -I 'https://maps.YOUR-DOMAIN/releases/russia-202609/fonts/Noto%20Sans%20Regular/1024-1279.pbf'
curl -I https://maps.YOUR-DOMAIN/releases/russia-202609/sprite.png
```

Проверить в браузере Network: все ресурсы карты идут на собственный origin, streets/buildings/номера видны на контрольном адресе. Проверить ETag/304, CORS, MIME и gzip. Метрика `taxi_routing_duration_seconds{status="success|cache_hit|error"}` публикует histogram; `_count{status="error"}` учитывает ошибки. `state.json`, `validated.json`, `activation.json`, `active.json` и API конфигурации показывают состояние и версии. Ошибки route-cache логируются с operation и не подменяют ответ маршрутизатора.

Результаты серверного пилота и остающиеся проверки: [maps-validation.md](maps-validation.md).

## Первичные источники

* [OSRM Route API](https://project-osrm.org/docs/v5.24.0/api/)
* [Tilemaker](https://github.com/systemed/tilemaker)
* [Pelias OSM importer](https://github.com/pelias/openstreetmap)
* [Pelias schema и ICU](https://github.com/pelias/schema)
* [Noto glyphs v2.0](https://github.com/openmaptiles/fonts/releases/tag/v2.0)
* [OSM attribution и ODbL](https://www.openstreetmap.org/copyright)
