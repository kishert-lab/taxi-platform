# GIS Backend Audit

Audit date: 2026-09-16. Scope: current repository worktree only. No production
host, Docker daemon, map-data volume, or mobile application was changed by this
audit.

## 1. Current Architecture

The repository contains a self-hosted GIS design. Its intended data flow is:

```text
Geofabrik OSM PBF (HTTPS + checksum)
  -> versioned release builder
  -> MBTiles / MVT, OSRM MLD graph, Pelias Elasticsearch index, local style assets
  -> map-origin (read only) -> nginx map-proxy -> application/device cache

taxi-platform -> routing.Service -> versioned OSRM
taxi-platform -> Pelias -> address search/reverse geocoding
taxi-platform -> Redis route cache -> route/estimate business responses
driver app -> taxi-platform WebSocket -> passenger driver marker
```

The backend does not render map images. It returns geographic data and a map
configuration; rendering is the client responsibility.

## 2. Existing Components

| Component | Current implementation | Network exposure / persistence |
| --- | --- | --- |
| Map data pipeline | `infrastructure/maps/manage.py` | Release root, default `/srv/taxi-maps` |
| Tile generator | Tilemaker, project `tilemaker.json` and `process.lua` | Produces `map.mbtiles` per immutable release |
| Map origin | `infrastructure/maps/server.py` WSGI service | Read-only release data; GET/HEAD only |
| Map proxy | `docker-compose.maps.yml` `map-proxy`, nginx | Binds `127.0.0.1:${MAP_HTTP_PORT:-8090}`; cache volume `map_proxy_cache` |
| Routing | `internal/routing` abstraction, OSRM client | Current release builder starts `maps-{version}-osrm` on external `taxi-maps` network |
| Routing cache | `internal/redis/route_cache.go` | Redis; key namespace contains OSRM URL, data version, profile, geometry, snap radius, and bounds |
| Geocoding | Pelias API, Elasticsearch, libpostal | Release builder creates an isolated index and Pelias service per release |
| Application map API | `internal/maps`, `MapHandler` | Exposes config, route, and reverse-geocoding through taxi-platform |
| Realtime driver data | existing WebSocket / Redis gateway | `driver.location_updated`; not embedded in tiles |

The primary production Compose files also contain an older Pelias stack. In
`docker-compose.prod.yml`, Pelias, Elasticsearch, and libpostal are internal.
The development `docker-compose.yml` publishes local Postgres, Redis, Pelias,
and Elasticsearch ports for development convenience.

## 3. Current Data Flow

`manage.py build <version>` downloads the configured Geofabrik PBF over HTTPS,
checks its checksum, and builds an unvalidated release. It creates:

- gzip-compressed MVT-compatible tiles in `map.mbtiles`;
- an OSRM MLD graph from the same PBF;
- a Pelias index from the same PBF;
- local `style.json`, sprites, glyphs, and attribution assets.

Validation checks MBTiles integrity and control layers, OSRM routing, Pelias
search/reverse responses, and local assets. Only a release with
`validated.json` is served. Activation writes backend environment overrides so
the backend uses the selected release's OSRM and Pelias services; it also checks
that `/api/v1/public/map/config` exposes the selected version.

`map-origin` serves only `/releases/{version}/...`. Tiles are read from MBTiles
using immutable SQLite access; assets are served from the release's `public`
directory. nginx owns CORS, cache, gzip, and reverse-proxy caching.

## 4. Existing API Endpoints

The following application-facing endpoints are registered by `MapHandler`:

| Endpoint | Access | Purpose |
| --- | --- | --- |
| `GET /api/v1/public/map/config` | public | Versioned style URL, bounds, zoom range, attribution, availability |
| `POST /api/v1/passenger/map/routes` | passenger JWT | Road route for exactly two coordinates |
| `GET /api/v1/passenger/address/reverse` | passenger JWT | Reverse geocoding through taxi-platform |
| `POST /api/v1/map/routes` | currently public | Same route handler; see gap G-1 |
| `GET /api/v1/geocoder/reverse` | currently public | Same reverse handler; see gap G-1 |
| `GET /api/v1/passenger/address/search` | passenger JWT | Address search through application geocoder service |
| `POST /api/v1/passenger/orders/estimate` | passenger JWT | Route metrics plus park tariff business estimate |

Taxi park estimate endpoints and driver order-route/history endpoints are
separate business contracts. The WebSocket contract already publishes
`driver.location_updated` to the passenger; no additional realtime channel is
required for map rendering.

## 5. Existing Map Format

Base map format is MVT-compatible PBF stored in MBTiles. The origin endpoint is
effectively:

```text
GET /releases/{version}/tiles/{z}/{x}/{y}.pbf
```

Supported tile zoom is 0 through 14. The current selected layers are `road`,
`building`, `housenumber`, `place`, `poi`, `water`, and `landuse`; this is a
deliberately reduced taxi-oriented schema rather than a full OSM export.

Map geometry and style are separated. The release includes a versioned
MapLibre-compatible `style.json`, sprites, and local Noto glyph PBFs. The
origin returns ETags and immutable one-year cache headers for existing release
assets. nginx additionally caches responses and permits only GET, HEAD, and
OPTIONS below `/releases/`.

No raster-tile pipeline was found. A route is currently represented as GeoJSON
`LineString`; it is valid but less compact than an encoded polyline.

## 6. Existing Routing Implementation

`internal/routing.Service` is the business boundary. `osrm.Client` implements
it with OSRM's driving route API, validates input coordinates, coverage bounds,
snap distance, response size, geometry, and route metrics. It returns route
geometry, meters, seconds, original/snapped points, data version, source, and
calculation time.

`cmd/api/routes.go` constructs this service from `TAXI_ROUTING_*` settings and
wraps it in Redis caching. Passenger orders, taxi-park estimates, and map route
API share the same abstraction. Price calculation remains in taxi-platform and
uses the route metrics with taxi-park tariffs.

## 7. Existing Geocoding Implementation

The intended release-specific geocoder is Pelias with Elasticsearch and
libpostal. `manage.py` builds an isolated index from the release PBF and probes
both `/v1/search` and `/v1/reverse` before validation. The backend calls Pelias
internally; the mobile client does not receive Elasticsearch, PostgreSQL,
libpostal, or Pelias credentials/endpoints.

The general geocoder configuration still includes optional Yandex and DaData
providers. They are not part of the self-hosted release pipeline, but enabling
them would introduce external per-request geocoding dependencies.

## 8. Ports, Networks, and Data

The release map proxy is the intended application-facing map origin. It binds
only to loopback in `docker-compose.maps.yml`; an external HTTPS proxy is still
needed to publish the configured `MAPS_PUBLIC_URL` safely.

Release containers use the external `taxi-maps` Docker network. The map origin,
OSRM, Pelias, Elasticsearch, and libpostal containers have no published ports
in generated release Compose. PostgreSQL, Redis, Elasticsearch, libpostal, and
Pelias must remain internal in production Compose.

Persistent GIS data is versioned under `/srv/taxi-maps/releases/{version}`:
PBF, MBTiles, OSRM graph, Pelias work files/index, generated assets, manifests,
and validation state. nginx cache is a separate Docker volume.

## 9. Problems and Gaps

G-1. `POST /api/v1/map/routes` and `GET /api/v1/geocoder/reverse` are public,
while passenger equivalents are JWT-protected. Confirm whether a deliberately
public web-map use case exists; otherwise remove or protect the public aliases.

G-2. The worktree contains two OSRM deployment models: a standalone `osrm`
service in `docker-compose.yml` reading `/srv/osrm/data`, and the versioned
release OSRM started by `manage.py` from `/srv/taxi-maps/releases/{version}`.
They can use different PBF versions. Do not run both as authoritative routing
sources. Select the release-managed model before deployment.

G-3. `docker-compose.prod.yml` retains an independent Pelias stack while the
release pipeline creates versioned Pelias/Elasticsearch services. This has the
same data-version divergence risk as G-2.

G-4. The release map proxy is loopback-only and its HTTPS termination/domain,
certificate, and allowed CORS origin are deployment prerequisites, not yet
proven by repository configuration.

G-5. `docs/maps-validation.md` records a Monaco pilot, not a Russia release.
Full Russia capacity planning, actual disk/RAM/Elasticsearch heap measurements,
and end-to-end client tests remain required.

G-6. Current route responses use GeoJSON. This is acceptable for short routes,
but an encoded-polyline response should be evaluated before high-volume mobile
use; preserve GeoJSON for compatibility/debugging if introduced.

G-7. Optional Yandex and DaData geocoding providers conflict with a strict
self-hosted-only runtime policy when enabled. Their defaults should remain off
until a separate product decision explicitly permits them.

## 10. Proposed Target Vector-Tile Architecture

Retain the current release pipeline as the target rather than introduce a new
stack:

```text
versioned OSM PBF
  -> Tilemaker -> immutable MBTiles/MVT
  -> read-only map-origin -> nginx HTTPS proxy/cache -> device tile cache

same versioned PBF
  -> OSRM MLD -> routing.Service -> taxi business APIs
  -> Pelias index -> taxi geocoding APIs
```

The style stays a separately versioned asset. Base map remains static and
cacheable; route geometry remains a REST response; driver position remains an
authorized WebSocket event; order state remains the existing REST/WebSocket
business flow.

## 11. Proposed Routing Boundary

Keep `routing.Service` as the sole taxi-platform dependency. OSRM is an
infrastructure implementation, never a mobile API. The application exposes
route/estimate results through documented handlers and may later add a compact
encoded-polyline field without exposing OSRM syntax or credentials.

One release identifier must bind tiles, style, OSRM graph, and Pelias index.
The existing `maps.data_version == routing.data_version` validation and
activation check are the correct base; the standalone Compose services need
reconciliation with this release model before activation.

## 12. Migration Plan

1. Decide whether the versioned release pipeline is the sole production GIS
   source. Recommended: yes.
2. Remove or disable duplicate standalone OSRM/Pelias deployment paths only
   after a tested activation/rollback procedure exists.
3. Define the HTTPS maps hostname and CORS origins; keep map management APIs
   private and expose only immutable release assets.
4. Build a regional pilot matching the actual operating area, measure it, then
   build Russia only with an approved capacity plan and retention budget.
5. Decide the authorization policy for public map route/reverse aliases.
6. Validate Expo/React Native renderer compatibility against the real style,
   MVT endpoint, glyphs, sprites, caching behavior, and Android 9 devices.
7. Only then consider a compact route encoding and mobile client adoption.

## 13. Resource Implications and Risks

The system trades external API billing for local CPU, RAM, disk, network, and
operations. Builds require temporary disk headroom plus retained releases;
OSRM MLD and Elasticsearch are the dominant memory/disk consumers. Build and
activation already provide locking, immutable validated releases, health probes,
and rollback, but production resource sizing has not been measured for Russia.

Operational metrics currently include routing latency/cache/error histogram
`taxi_routing_duration_seconds`. Add or verify dashboard coverage for map proxy
request rate, cache hit ratio, response bytes, origin errors, OSRM errors,
Pelias latency/errors, disk free space, container CPU/RAM, and network traffic
before production rollout.

## Audit Conclusion

The repository already has the required self-hosted vector-tile, routing, and
geocoding direction. The minimal path from CURRENT to TARGET is to complete and
operate the existing versioned release pipeline, not to add another GIS stack.
This audit intentionally makes no infrastructure, routing-engine, or public API
changes.
