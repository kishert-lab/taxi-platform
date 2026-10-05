# Driver transfers by invitation

Apply migration `000040_driver_park_invites` before starting this API version.
The previous API version cannot write the new park-scoped balance schema. Stop
the old API during migration and start the matching new binary afterwards.
Migration `000039` already present in the workspace is not modified by this feature.

`TAXI_DRIVER_INVITE_TTL` controls lifetime (default `168h`). Existing JWTs remain
valid: they contain user identity, never a park; each request resolves the driver
and their current park from PostgreSQL. An unassigned driver can log in to receive
an invitation but cannot go online without a park and the existing verification.

## API

All routes use `/api/v1` and access-token authentication.

| Route | Role / request |
| --- | --- |
| `POST /taxi-park/driver-invites` | Park or active dispatcher; `{ "phone": "+79991234567" }` |
| `GET /taxi-park/driver-invites` | Own sent invitations, newest 200 |
| `POST /taxi-park/driver-invites/:id/cancel` | Own pending invitation |
| `GET /driver/park-invites` | Own invitations, newest 200; also the persistent notification inbox |
| `POST /driver/park-invites/:id/accept` | Explicit driver confirmation |
| `POST /driver/park-invites/:id/decline` | Explicit driver rejection |
| `PUT /driver/push-token` | `{ "token": "...", "platform": "android" }`; also ios/web |
| `DELETE /driver/push-token` | `{ "token": "..." }` before logout |

Responses use the existing `data` envelope. An invitation includes `id`,
`driver_id`, source/destination park IDs and names, status, creator and lifecycle
timestamps. Source park identity is omitted from park-facing responses. Driver
creation retains its original response; a duplicate phone returns HTTP 409 with
code `DRIVER_ALREADY_REGISTERED`, which opens an invitation offer in the panel.

Expired invitations are reported as `expired` immediately. Mutation materializes
expiry before further processing. Creating the same live invitation is idempotent.
All lifecycle mutations lock the driver before locking invitations. Accepting
an invitation cancels the other pending invitations with reason
`driver_transferred_to_another_park`.

The driver must be offline and have no unfinished or preassigned orders. There is
no separate shift entity in the existing project. Transfer preserves driver/user
IDs and phone, closes/opens park history, detaches old park cars, clears park-local
commission/comment, and requests verification by the new park. Existing orders
and financial records are not rewritten. A return to a former park restores that
park's account; a first visit starts at zero. Old parks can pay their outstanding
balances. Delayed settlement uses the order's stored park, not the driver's park.

Redis/WebSocket publishes `driver.park_invite.created|accepted|declined|cancelled`.
The existing Firebase provider sends creation pushes when configured and a driver
token is registered. Delivery failure is logged and does not undo a committed
invitation; the inbox is refreshed after login/reconnect/resume and periodically.
No external message delivery is claimed to be exactly-once. Park dashboards
receive lifecycle events. Audit uses the existing `finance_audit_events` table.

## Clients and verification

Panel: `Q:\taxi-web`, driver Flutter app: `D:\develoop\taxi-driver`.
Both display accept/decline actions without changing park when merely opening an
invitation. The Flutter app refreshes profile, cars and balance after acceptance,
discards stale pre-transfer profile/balance responses, and registers FCM tokens.

Run `go test ./...`. For the PostgreSQL integration tests, create an **isolated**
database and apply all `migrations/*.up.sql` in filename order, one transaction
per file (for example `psql -1 -v ON_ERROR_STOP=1 -f ...`). Then:

```sh
TAXI_INVITE_TEST_DATABASE_URL='postgres://.../disposable_database?sslmode=disable' \
  go test -race ./internal/repository -run TestDriverParkInvitesIntegration -count=1
```

Tests insert their own fixtures. Do not target a working database. Coverage includes
duplicate identities/invitations, notification persistence, transfer history,
balances and payouts, delayed settlement, old JWT profile resolution, decline,
cancel, expiry, ownership, active work, concurrency, and transaction rollback.
Client checks: `npm run build && npm test`; `flutter analyze && flutter test`.
