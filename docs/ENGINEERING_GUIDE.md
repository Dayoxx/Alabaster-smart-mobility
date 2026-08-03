# Engineering Guide — Ilorin Transport Intelligence Platform

This is the guide every developer on this project follows — backend, Flutter, and React. The **master architecture doc** (`transport-intelligence-roadmap.html`) explains *what* we're building and *why*. This guide explains *how to write the code* so that a growing team stays consistent, and a security-critical payments system doesn't accumulate inconsistent patterns as more people touch it.

If something here conflicts with the architecture doc, the architecture doc wins on *design decisions* (schema, flows, sequencing) — this guide wins on *how you write the code that implements those decisions*.

---

## 0. Before you write a line of code

**Golden rules — these are non-negotiable, not style preferences:**

1. **Never touch `*sql.DB` directly in a handler.** Always go through a repository interface. This is the dependency injection seam that makes the whole system testable — breaking it once means every test after that has to hit a real database.
2. **Never store mutable state in a package-level variable.** The original prototype had `var Users = []User{}` — a global slice mutated across goroutines with zero locking. It was a real, live data race. It's also the exact pattern that makes horizontal scaling (Layer 11) impossible later. If you're tempted to write `var something = []Thing{}` at package scope, stop — that state belongs in Postgres or nowhere.
3. **Every query is parameterized.** `$1, $2...` — never string concatenation, never `fmt.Sprintf` into SQL, ever.
4. **Secrets never enter source control.** Environment variables only. If you paste an API key into a file, you've already failed this rule even if you delete it before committing — rotate it.
5. **Payment and session data is never cached.** Fare tables and dashboard aggregates can be. Balances, sessions, and trip state cannot.
6. **Every external call (Moniepoint, VSP, NIMC) has a timeout.** No exceptions.

If you're not sure whether something violates one of these, ask before you PR it.

---

## 1. Frontend Foundations

### Flutter (Driver / Passenger / Student app)

**Folder structure is feature-first, not layer-first.** Don't create a global `screens/` folder with everything dumped in — each feature owns its own `screens/`, `widgets/`, `providers/` internally.

```
lib/
├── core/           # config, constants, network client, security (device-bound key)
├── data/           # models, repositories, datasources (remote + local)
├── features/
│   ├── auth/
│   ├── driver_home/
│   │   ├── session/
│   │   ├── nfc_tap/
│   │   ├── fallback/
│   │   ├── gis_tracking/
│   │   ├── bank_details/
│   │   ├── transaction_history/
│   │   └── vehicle_mode/keke/
│   ├── passenger_home/
│   ├── student_home/
│   ├── admin_home/       # mobile, monitoring-only — never build privileged actions here
│   ├── ops/               # staff-only, HQ
│   └── agent_ops/         # staff-only, field
├── presentation/
└── state/
```

**Naming conventions:**
- Screens: `PascalCase` ending in `Screen` — `ShiftStartScreen`, not `shift_start_page` or `StartShift`.
- Riverpod providers: end in `Provider` — `sessionStateProvider`, `activeRouteProvider`.
- Repositories: end in `Repository` — `SessionRepository`, matching the Go-side naming exactly so a dev can find the corresponding backend interface without guessing.

**Adding a new feature — the checklist:**
1. Create the feature folder under `features/`.
2. Define the repository interface in `data/repositories/` *before* writing any UI — this mirrors the Go-side discipline of designing the interface first.
3. Wire a Riverpod provider for any state that outlives a single widget.
4. Build the screen last, against the provider — never call the repository directly from a widget.

**✅ Do**
- Cache route/destination lists client-side for the shift (per the Fare Engine section) — don't refetch before every tap.
- Use `flutter_secure_storage` for anything touching the device-bound key or cached biometric template. Never `SharedPreferences`.

**❌ Don't**
- Don't put business logic (fare calculation, session validity checks) in a widget. Widgets render state; repositories and providers own logic.
- Don't call `dio` directly from a screen. Every network call goes through `core/network`'s signing interceptor.

### React (Admin dashboard)

**TypeScript, not JavaScript — mandatory, not a preference.** Interfaces should mirror the Go structs they represent, field for field:

```typescript
// Go: type Session struct { ID string; DriverID string; ExpiresAt time.Time; Status string }
interface Session {
  id: string;
  driverId: string;
  expiresAt: string; // ISO 8601 — Go's time.Time marshals to this
  status: 'active' | 'expired' | 'flagged';
}
```

If a Go struct changes, the corresponding TS interface changes in the same PR, not a follow-up one.

**✅ Do**
- Keep privileged actions (rebind, reconciliation, fare edits) behind explicit confirmation dialogs — these hit `UpdateFare`, `RebindTerminal`, etc. and are logged with an admin ID. Never fire them on a single click.

**❌ Don't**
- Don't fetch dashboard aggregate data (heatmap, corridor stats) on every render. Respect the caching layer (Layer 10) — these queries are expensive on purpose-designed short TTLs, not free.

---

## 2. APIs & Backend Logic (Go)

**Package structure** (already established, don't deviate):

```
cmd/api/main.go            # package main — wiring only, no business logic
internal/models/           # package models — plain structs, no methods beyond simple helpers
internal/db/                # package db — connection setup, WithTx helper
internal/repository/        # package repository — interfaces + Postgres implementations
```

**The handler pattern — every handler follows this exact shape:**

```go
func (app *App) someHandler(c *gin.Context) {
    var input struct {
        Field string `json:"field" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // call the repository interface — never a concrete *sql.DB
    result, err := app.SomeRepo.DoThing(c.Request.Context(), input.Field)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not do thing"})
        return
    }

    c.JSON(http.StatusCreated, result)
}
```

**Rules:**
- Input structs are defined *inline*, inside the handler, unless reused elsewhere. Don't create a top-level `SomeThingRequest` struct for a one-off.
- Every handler takes `c.Request.Context()` through to the repository call — this is what makes a dropped client connection cancel the underlying query instead of running it to completion uselessly.
- Error messages returned to the client are generic (`"could not do thing"`). Real error detail goes to structured logs (Layer 12), never to the response body — a verbose error message is a reconnaissance gift to anyone probing the API.

**Repository pattern — the actual DI seam:**

Every repository struct is built against the `Querier` interface (satisfied by both `*sql.DB` and `*sql.Tx`), never `*sql.DB` directly:

```go
type Querier interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

This is what lets the exact same repository code run standalone *or* inside a `db.WithTx(...)` transaction — see the tap-flow fix for the reference pattern. If you're writing a handler that does more than one write and they need to succeed or fail together, it belongs inside `WithTx`, full stop.

**✅ Do**
- Write a fake implementation of any repository interface you add, even before you write a test for it. If you can't picture the fake, the interface is wrong.

**❌ Don't**
- Don't add a new package-level `var` for anything that isn't a genuine constant.
- Don't call `vsp.VerifyFingerprint` or `moniepoint.Transfer` directly from a handler — always through the injected interface (`IdentityVerifier`, `PaymentProcessor`), so tests can substitute a fake.

**The convention for any handler that overwrites a value someone might contest later:** this has now shown up independently in `UpdateFare`, `RebindTerminal`, and card batch reconciliation — that's not a coincidence, it's a real pattern. Any privileged write that changes a value (not just creates a new row) follows this shape, not a bare `UPDATE`:

```go
func UpdateSomething(id string, newValue T, adminID, reason string) error {
    return db.WithTx(func(tx) error {
        oldValue := db.GetCurrentValue(tx, id)
        if err := db.SetValue(tx, id, newValue); err != nil { return err }
        return db.InsertChangeLog(tx, ChangeLog{
            EntityID: id, OldValue: oldValue, NewValue: newValue,
            ChangedBy: adminID, Reason: reason, ChangedAt: time.Now(),
        })
    })
}
```

Old value captured, new value applied, both logged, atomically. If you're writing a new admin action that changes something rather than adding something, use this shape by default — don't invent a fourth slightly different version of it.

---

## 3. Database & Storage

**Migrations:**
- Numbered sequentially: `0001_init.sql`, `0002_terminal_api_key.sql`, never reordered or renumbered after merge.
- One migration = one logical change. Don't bundle an unrelated schema tweak into a migration named for something else.
- Every migration is additive by default (`ADD COLUMN IF NOT EXISTS`). Destructive migrations (drop column, drop table) need a second reviewer and a note on how existing data is handled.

**Schema conventions:**
- Primary keys: `UUID`, always.
- Columns: `snake_case`.
- Timestamps: `TIMESTAMPTZ`, never bare `TIMESTAMP` — timezone-naive timestamps are a bug waiting to surface the first time this runs across regions.
- Foreign keys: `<referenced_table_singular>_id` — `driver_identity_id`, not `driver_id_fk` or `fk_driver`.
- Booleans: `is_active`, `is_verified` — prefixed, never bare `active`.

**PostGIS:**
- Geometry columns are `geometry(Point, 4326)` or `geometry(LineString, 4326)` — 4326 (WGS84) always, don't mix SRIDs.
- Spatial queries use `ST_DWithin` with `::geography` casts for real-world distance, not raw `ST_Distance` on `geometry` — the latter gives you degrees, not meters, and it's a subtle bug that only shows up when someone eyeballs the numbers and they look wrong.

**Naming trap to know about — `terminal` vs `terminal_location`:**
These mean genuinely different things and the confusion is easy to reintroduce in code even after it's been fixed in the docs. `terminal` is a driver's physical device (`hardware_serial`, bound to one driver). `terminal_location` is a place — Tanke, Challenge. If you're naming a variable, a function parameter, or a column and you write bare `terminal` or `terminalID`, stop and ask which one you mean — then name it `deviceTerminalID` or `terminalLocationID` explicitly. This exact ambiguity already caused real confusion in the architecture doc's own roadmap section; don't let it happen again in code where it's harder to spot.

---

## 4. Auth & Permissions

*(Design here is now thorough — session middleware for the four customer logins, plus the full admin access model below — but almost none of it is in the real codebase yet, which still only has the stopgap terminal-key check from the `/tap` fix. Whoever picks this up next has a clear spec to build against, not a blank page.)*

**Every protected route needs middleware, not an inline check.** Don't write `if session.Role != "driver" { ... }` inside a handler body — that pattern gets copy-pasted inconsistently as routes grow. Middleware, applied at route registration:

```go
router.POST("/tap", RequireRole("driver"), app.tapHandler)
router.POST("/admin/rebind", RequireRole("admin"), RequireAudit(), app.rebindHandler)
```

**RBAC on the admin side is ops vs. super-admin** — don't collapse this into a single "is admin" boolean. Terminal rebind and card batch reconciliation need `super_admin`; read-only dashboard views need only `ops`. The field is `identity.admin_tier`, checked the same middleware way as any other role:

```go
router.POST("/admin/rebind", RequireAdminTier("super_admin"), RequireAudit(), app.rebindHandler)
```

**There is no public admin registration endpoint. Ever.** Don't add one, even a "temporary" one for testing — this is the single easiest way to accidentally let anyone self-grant access to live driver GPS and fare-editing power. The only two ways an admin account gets created:
1. The first account, via a one-time deployment script, not through the running app.
2. Every account after that, through `InviteAdmin` — callable only by an existing `super_admin`, and itself a logged, audited action like any other privileged one.

**WebAuthn/passkey login — use a real library, don't hand-roll the verification.** The challenge-response signature check in `VerifyPasskeyLogin` has known, easy-to-get-wrong edge cases (challenge replay, origin binding) that a maintained library handles correctly. This is exactly the kind of security-critical code where "I'll just implement the spec myself" is how a subtle vulnerability gets shipped — use `go-webauthn` or equivalent, don't write the signature verification from scratch.

**Session tokens are never logged**, not even at debug level. If you need to debug a session issue, log the session *ID*, never the token itself.

---

## 5. Hosting & Deployment

**Dockerfile is multi-stage** — build stage compiles the Go binary, final stage is a minimal image with just the binary. Don't ship the Go toolchain in production images.

**Environment variables** are the only source of configuration — no config files with environment-specific values checked into git. Naming: `SCREAMING_SNAKE_CASE`, prefixed by concern where it helps clarity (`DATABASE_URL`, `MONIEPOINT_API_KEY`, `VSP_ENDPOINT`).

**Deployment checklist, every time:**
- [ ] Migrations run *before* the new binary deploys, never after
- [ ] `/health` returns 200 before traffic is routed to the new instance
- [ ] Old instance drains (graceful shutdown, 15s window) before it's killed

---

## 6. Cloud & Computing

**Three environments, minimum: dev, staging, production.** Never test against production data. Staging uses Moniepoint's sandbox credentials, never live ones — mixing these up is how real money moves during a test.

**Secrets are environment-scoped** — a staging `DATABASE_URL` and a production one are different values in different places, never the same secret reused "to save time."

---

## 7. CI/CD & Version Control

**Branching**: trunk-based. Short-lived feature branches off `main`, merged via PR, no long-running `develop` branch accumulating drift.

**Commit messages**: describe the *why*, not just the *what* — `"Add replay protection to /tap"` not `"update main.go"`.

**Every PR must:**
- [ ] Pass CI (`go vet`, `go build ./...`, tests, secret scan)
- [ ] Include a test for any change to session, auth, or payment logic — no exceptions, this is where the fake repository pattern earns its keep
- [ ] Note in the description if it touches a Golden Rule from Section 0

**CI pipeline stages**: lint/vet → build → test → (on merge to `main`) build Docker image → deploy.

---

## 8. Security & Row-Level Security

The full checklist lives in the master architecture doc's Security Checklist section — read it. The rules that matter most *while writing code*, not just planning it:

- Redaction happens in logging middleware, once, centrally. Don't remember to redact per-call-site — you will eventually forget, and it only takes one miss.
- Row-level security (where it's enabled, T3+) is defense-in-depth against a bug in your own authorization logic — write your Go-side authorization correctly *first*. RLS is a safety net, not a substitute for getting the application logic right.
- Any PR touching `AuthorizeTrip`, `StartDriverSession`, device binding, or payout resolution needs a second reviewer, always, regardless of how small the change looks.

---

## 9. Rate Limiting

New endpoint, new rate limit — don't assume the global default covers it. Pattern:

```go
router.POST("/tap", RateLimitByTerminal(), app.tapHandler)
```

Per-terminal limits for driver-facing endpoints, per-IP for public ones. If you're adding an endpoint that doesn't fit either pattern, ask before shipping it unlimited.

---

## 10. Caching & CDN

**The only things that get cached**: dashboard aggregate queries (heatmap, corridor stats, revenue tickers) — short TTL, 30–60 seconds. Route/stop lists client-side per shift.

**Never cached, ever**: session state, account balances, fare calculations for an in-progress trip, anything from `AuthorizeTrip`.

If you're not sure whether something is safe to cache, the answer is no until someone with full context on the payment flow says otherwise.

---

## 11. Load Balancing & Scaling

**Every handler must be stateless** — nothing about a request's outcome should depend on which specific server instance handled it. This is Golden Rule #2 from Section 0, restated here because it's the direct reason horizontal scaling works at all.

**`/ws/map` is the one genuine exception to think about**: once there's more than one API instance, WebSocket connections need a shared pub/sub (Redis) to fan out position updates across instances — a client connected to instance A needs to see updates that originated on instance B. Don't build this before it's needed (single instance is fine through T3), but don't accidentally paint yourself into an architecture where it's hard to add either.

---

## 12. Error Tracking & Logs

**Structured logging, always** — `log/slog`, key-value fields, never a bare formatted string you'll have to regex later.

```go
slog.Error("trip authorization failed", "terminal_id", terminalID, "error", err)
// not: log.Printf("failed to authorize trip for terminal %s: %v", terminalID, err)
```

**What never appears in a log line, at any level**: fingerprint template hashes, session tokens, full account numbers, Moniepoint API keys. If a struct being logged might contain one of these, log specific safe fields instead of the whole struct.

**Sentry** (or equivalent) catches unhandled errors — but don't use it as a substitute for handling errors you already know how to handle. An `if err != nil` you're not sure what to do with is a code smell, not a Sentry ticket.

---

## 13. Availability & Recovery

**Every external call gets a timeout** — Moniepoint, VSP, NIMC, anything crossing a network boundary you don't control:

```go
ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
defer cancel()
```

No exceptions. An external service hanging shouldn't be able to hang your whole request pipeline.

**`/health` checks real dependencies**, not just "the process is running":

```go
router.GET("/health", func(c *gin.Context) {
    if err := app.Conn.PingContext(c.Request.Context()); err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unreachable"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
})
```

**Backups are verified, not just scheduled.** A cron job that's been silently failing for two months isn't a backup — if you're touching backup infrastructure, confirm a restore actually works, don't just confirm the job runs.

**Manual payout retries reuse the original idempotency key — never a fresh transfer.** If ops needs to resend a payout because a driver says they didn't receive it, that action calls `moniepoint.Transfer` with the *original trip's nonce* as the idempotency key, exactly like the automated path. A fresh ad-hoc transfer with no idempotency key is how a driver ends up paid twice for one trip — this is a real operational failure mode, not a hypothetical one, and it happens through good intentions ("let me just fix this for them quickly"), not carelessness.

---

## Appendix A: PR Checklist (copy into every PR description)

- [ ] Follows the folder/naming conventions for its layer
- [ ] No new package-level mutable state
- [ ] All queries parameterized
- [ ] Secrets via environment variables only
- [ ] Tests included if touching session, auth, or payment logic
- [ ] Structured logging, no sensitive fields logged
- [ ] External calls have timeouts
- [ ] Second reviewer tagged if touching a Golden Rule area

## Appendix B: New Developer Onboarding

1. Read the master architecture doc (`transport-intelligence-roadmap.html`) — the schema, the pseudocode flows, the security fixes and *why* they exist.
2. Read this guide, Section 0 twice.
3. Set up dev environment against **staging**, never production, and confirm `DATABASE_URL` points where you think it does before running anything.
4. Your first PR should be small and touch exactly one layer — this is a better way to learn the conventions than reading about them.