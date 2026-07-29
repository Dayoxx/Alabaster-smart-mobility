markdown
# Transportation-model — Postgres migration (step 2)

This replaces the in-memory `[]User{}` / `[]Terminal{}` / etc. slices with
Postgres, wired through repository interfaces so handlers depend on
interfaces, not a concrete database — this is the dependency injection seam
for testing later.

## What changed from the original single-file version

- **Bug fix**: the terminal registration duplicate check was comparing
  `TerminalType` instead of `DeviceSerial` — meant a second Keke terminal
  could never be registered once the first one existed. Fixed in
  `FindByDeviceSerial`.
- **Removed** the unused transit types (train/plane/tram/monorail) from the
  valid-types check — this platform is Keke and bus only.
- **Split across real packages** to match this project's actual structure —
  everything used to live in one `main.go`; it doesn't anymore.
- JSON request/response shapes are unchanged, so nothing calling this API
  needs to change.

## Where everything lives

cmd/api/main.go → package main — entry point, wires
everything together, HTTP handlers
internal/models/models.go → package models — User, PassengerAccount,
Terminal, Trip
internal/db/db.go → package db — Postgres connection setup
internal/repository/repository.go → package repository — interfaces
(the dependency injection seam)
internal/repository/postgres.go → package repository — Postgres
implementations, parameterized queries
throughout (no string-concatenated SQL)
migrations/0001_init.sql → schema matching the models exactly


Handlers only ever depend on the `repository` interfaces, never on
`*sql.DB` directly — in production `main.go` wires in the real Postgres
implementations, in a test you'd wire in a fake that returns canned data.

## Setup

```powershell
# 1. install the Postgres driver and sync go.mod/go.sum
go get github.com/lib/pq
go mod tidy

# 2. create the database and run the migration
createdb transport
psql transport -f migrations/0001_init.sql

# 3. create a least-privilege role instead of using your superuser locally
#    (uncomment the two lines at the bottom of the migration file, or run manually)

# 4. set the connection string — never hardcode this
$env:DATABASE_URL = "postgres://transport_api:change-me@localhost:5432/transport?sslmode=require"

# 5. build everything (checks all packages, not just one file)
go build ./...

# 6. run the actual API
go run ./cmd/api
```

Note `go build ./...` and `go run ./cmd/api` — not `go run .` or
`go build main.go`. With real subpackages, those only look at a single
file/folder and will report every type in `models`/`db`/`repository` as
undeclared, even though the code is correct.

## What this does NOT yet do (on purpose — next steps)

- No `session` table, so `/tap` still authorizes against any existing
  terminal with no check that a driver is actually signed in. That's the
  next thing to build, not this migration.
- No `hardware_serial`/`bound_driver_identity_id` device-binding fields yet.
- No fingerprint/NIN/agent/card_batch tables — those come from the wider
  identity architecture, once this base is solid.

Each of these maps to a specific pseudocode block in the master roadmap doc
— this migration is just T0's "get off in-memory storage" piece.