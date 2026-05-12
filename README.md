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
make build
make test
make lint
make swagger
```

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
internal/middleware     HTTP middleware
internal/repository     PostgreSQL repositories
internal/redis          Redis queues, locks, offers, presence
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
