# Проверка картографической инфраструктуры

Проверка проведена 10 сентября 2026 года на сервере `192.168.0.50`. Это
пилот Monaco, а не неполная публикация данных России.

| Проверка | Результат |
| --- | --- |
| PBF → векторные тайлы | Успешно; на zoom 14 обнаружены `road` 3748, `building` 1658, `housenumber` 150, `poi` 714. |
| Автомобильный граф OSRM MLD | Успешно; контрольный маршрут вернул `Ok`, расстояние, время и GeoJSON. |
| Импорт Pelias | Успешно; индекс содержит 2315 документов. Контрольные `/v1/search?text=Monaco` и `/v1/reverse` вернули результаты. |
| Локальные assets | Успешно: `style.json`, `sprite.png`, локальный Noto Sans glyph с кириллицей. Стиль не содержит внешних URL. |
| Origin + nginx | Успешно: 200, корректные `application/json`, `application/x-protobuf`, `image/png`, ETag, immutable Cache-Control, CORS. |
| MapLibre | Успешно в headless Chrome: `MAPLIBRE_OK`, локальные tiles, sprite и glyphs загружены через proxy. |
| Backend | `go test ./...`, `go test -race` для map/routing/cache/pricing пакетов и `go build ./cmd/api` прошли. Swagger regenerated. |

Пилотный набор расположен вне репозитория в
`/media/d/taxi-maps-pilot/releases/pilot-20260909b`; он не активирован для
production backend и не содержит данных России. Его размер после проверки —
143 267 324 байта. Значение нельзя экстраполировать на Россию: перед сборкой
России необходимо выполнить отдельный capacity planning, задать
`min_free_bytes` на основании измерений и обеспечить место минимум для
активного, предыдущего и нового release.

Пока не выполнено:

* полная сборка и контрольные запросы по России;
* публикация immutable map-origin image в production registry;
* установка реального HTTPS домена, сертификата и допустимого CORS origin;
* активация backend image с изменениями и проверка с реальными mobile/web
  клиентами;
* измерение времени, RAM, диска и Elasticsearch heap для российского PBF;
* проверка бизнес-тарифов и прав на production данных.

Пилот сначала выявил две конфигурационные проблемы: обычный Elasticsearch не
содержал ICU-анализатор Pelias, а каталог LevelDB импортера имел неверного
владельца. Они устранены в воспроизводимом pipeline выбором pinned
`pelias/elasticsearch` и созданием `pelias-work` с UID/GID импортера.
