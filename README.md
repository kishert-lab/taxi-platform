# Taxi Platform

Production-grade backend for a taxi platform focused on small cities and rural areas. The service is written in Go and follows Clean Architecture, DDD boundaries, repository pattern, explicit application services, PostgreSQL/PostGIS persistence, Redis-backed realtime state, and asynchronous dispatch workers.

## Stack

- Go 1.24+
- Gin
- PostgreSQL + PostGIS
- Redis
- WebSocket-ready realtime gateway interfaces
- JWT access/refresh tokens
- pgx
- golang-migrate
- Swagger/OpenAPI via swaggo
- zap logger
- Docker / Docker Compose
- Immutable legal document versioning
- Taxi park branding, settings, and park tariffs

## Run With Docker Compose

```bash
docker compose up --build
```

Services:

- API: `http://localhost:8080`
- PostgreSQL/PostGIS: `localhost:5432`
- Redis: `localhost:6379`
- Swagger: `http://localhost:8080/swagger/index.html`

## Migrations

Docker Compose runs migrations automatically through the `migrate` service.

Manual migration commands:

```bash
make migrate-up
make migrate-down
```

Override database URL when needed:

```bash
DATABASE_URL='postgres://taxi:taxi_password@localhost:5432/taxi?sslmode=disable' make migrate-up
```

## Backend

```bash
make run
make admin CMD="create-taxi-park --phone +79990000000 --email park@example.com --city-id <uuid> --name 'City Taxi' --accept-documents"
make build
make build-admin
make test
make lint
make swagger
```

Swagger is served at:

```text
http://localhost:8080/swagger/index.html
```

## API Surface

All application endpoints are mounted under `/api/v1`.

### Auth and Legal

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/email/send-code`
- `POST /auth/email/verify`
- `POST /auth/verify-code`
- `POST /auth/refresh`
- `POST /auth/logout`
- `GET /public/legal/privacy-policy`
- `GET /public/legal/terms`
- `GET /public/legal/consent`

### Passenger Mobile API

- `POST /passenger/profile`
- `GET /passenger/profile`
- `PATCH /passenger/profile`
- `POST /passenger/profile/photo`
- `POST /passenger/orders/estimate`
- `POST /passenger/orders`
- `GET /passenger/orders/current`
- `GET /passenger/orders/history`
- `GET /passenger/orders/{id}`
- `POST /passenger/orders/{id}/cancel`
- `POST /passenger/orders/{id}/rate`

Passenger and driver profile responses include `photo_url`, `rating`, and `ratings_count`.

### Driver Mobile API

- `GET /driver/profile`
- `PATCH /driver/profile`
- `POST /driver/profile/photo`
- `POST /driver/online`
- `POST /driver/offline`
- `POST /driver/location`
- `POST /driver/location/batch`
- `GET /driver/orders/current`
- `GET /driver/orders/history`
- `POST /driver/orders/{id}/accept`
- `POST /driver/orders/{id}/reject`
- `POST /driver/orders/{id}/arrived`
- `POST /driver/orders/{id}/start`
- `POST /driver/orders/{id}/complete`
- `POST /driver/orders/{id}/rate-passenger`
- `GET /driver/balance`
- `GET /driver/transactions`

### Taxi Park API

- `GET /taxi-park/settings`
- `PATCH /taxi-park/settings`
- `GET /taxi-park/tariffs`
- `POST /taxi-park/tariffs`
- `PATCH /taxi-park/tariffs/{id}`
- `GET /taxi-park/balance`
- `GET /taxi-park/drivers`
- `GET /taxi-park/orders`
- `GET /taxi-park/transactions`

Taxi park settings include branding fields such as display name, logo URL, primary/secondary colors, support contacts, legal details, payment toggles, commission override, and timeout settings.

### Admin API

- `GET /admin/finance/overview`
- `GET /admin/legal/documents`
- `POST /admin/legal/documents`
- `POST /admin/legal/documents/{id}/activate`
- `POST /admin/legal/documents/{id}/deactivate`

Legal documents are append-only versions. Publishing a new legal text creates a new row; accepted versions remain auditable.

### Realtime

- `GET /ws`

The WebSocket endpoint accepts JWT through the `Authorization` header or `token` query parameter for mobile fallback.

## Environment Variables

Use `.env.example` as the documented baseline.

| Variable | Description |
| --- | --- |
| `APP_ENV` | Application environment: `local`, `docker`, `production` |
| `HTTP_PORT` | Public HTTP port |
| `DB_HOST` | PostgreSQL host |
| `DB_PORT` | PostgreSQL port |
| `DB_USER` | PostgreSQL user |
| `DB_PASSWORD` | PostgreSQL password |
| `DB_NAME` | PostgreSQL database |
| `REDIS_HOST` | Redis host |
| `REDIS_PORT` | Redis port |
| `JWT_SECRET` | Access token signing secret |
| `JWT_REFRESH_SECRET` | Refresh token signing secret |
| `WS_READ_BUFFER` | WebSocket read buffer size |
| `WS_WRITE_BUFFER` | WebSocket write buffer size |
| `DISPATCH_INITIAL_RADIUS` | Initial dispatch radius in meters |
| `DISPATCH_MAX_RADIUS` | Maximum dispatch radius in meters |
| `DISPATCH_TIMEOUT_SECONDS` | Offer timeout in seconds |

The current Go config loader uses `TAXI_` prefixed variables for structured config fields, for example `TAXI_DATABASE_HOST`, `TAXI_REDIS_HOST`, and `TAXI_JWT_ACCESS_SECRET`.

## Project Structure

```text
cmd/api                 API bootstrap
configs                 YAML and ENV configuration
docs                    generated Swagger files
migrations              PostgreSQL/PostGIS migrations
internal/auth           auth, registration, consent, authorization application logic
internal/dispatch       asynchronous dispatch service and workers
internal/domain         pure domain models, statuses, permissions, value objects
internal/dto            request/response DTOs
internal/finance        commission settlement, financial transactions, balances
internal/legal          legal documents and user document acceptance
internal/middleware     HTTP middleware
internal/repository     PostgreSQL repositories
internal/redis          Redis queues, locks, offers, presence
internal/taxipark       taxi park settings, branding, and tariffs
internal/transport      HTTP handlers
pkg/logger              zap logger factory
```

## Development Workflow

Branch strategy:

- `main`: stable production-ready code only
- `develop`: integration branch
- `feature/auth`
- `feature/dispatch`
- `feature/ws`
- `feature/orders`
- `feature/postgis`
- `feature/driver-app-api`
- `feature/passenger-app-api`

Use conventional commits:

```text
feat(dispatch): add radius expansion logic
fix(ws): reconnect handling for drivers
refactor(order): split repository and service layer
test(dispatch): cover redis accept lock
docs(readme): document local startup
```

Large modules should be split into separate pull requests with:

- change summary
- migration notes
- API changes
- breaking changes
- test evidence

## Personal Data Consent

Registration requires explicit consent under Russian Federal Law No. 152-FZ:

- personal data processing consent
- privacy policy version
- terms acceptance
- terms version
- timestamp
- IP address
- User-Agent

The backend validates consent even if the frontend sends malformed data. Consent audit events are persisted in `user_consent_events` for later audit, revocation, document version updates, and personal data deletion workflows.

## Legal Documents

Legal text is stored in PostgreSQL in `legal_documents` with immutable versions and active-version selection per document type and language.

Supported document types:

- `privacy_policy`
- `terms_of_service`
- `driver_agreement`
- `taxi_park_agreement`
- `consent_personal_data`

User acceptance is stored in `user_document_acceptance` with:

- `user_id`
- `document_id`
- `document_version`
- `accepted_at`
- `ip`
- `user_agent`

This keeps old accepted versions available for audit even after a new active document version is published.

## Taxi Park Settings

Taxi parks can maintain own settings in `taxi_park_settings`:

- branding: display name, short name, logo, colors
- support contacts: phone, email, website
- legal information: legal name, address, INN, OGRN
- commercial settings: commission override, minimum order price
- operations: cancellation and driver arrival timeout
- payment toggles: cash, card, transfer

Park-owned tariff customization is stored in `taxi_park_tariffs`, including base price, per-km price, per-minute price, minimum price, and future-ready fixed routes JSON.

## Admin Console

Privileged operational commands live in `cmd/admin` and use the same database configuration as the API.

Create a taxi park with an owner account and autogenerated password:

```bash
go run ./cmd/admin create-taxi-park \
  --phone +79990000000 \
  --email park@example.com \
  --city-id 11111111-1111-1111-1111-111111111111 \
  --name "City Taxi" \
  --legal-name "ООО City Taxi" \
  --tax-id 1234567890 \
  --commission-percent 1.00 \
  --accept-documents
```

Reset password by phone and role:

```bash
go run ./cmd/admin reset-password --phone +79990000000 --role taxi_park
```

Both commands generate a secure password when `--password` is omitted. Password reset revokes active refresh tokens for the selected user role.
