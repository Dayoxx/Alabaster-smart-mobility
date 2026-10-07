# Alabaster Smart Mobility — Master Engineering Guide

**This file replaces:** `ENGINEERING_GUIDE.md` (main), the additional card/payment implementation guide, the MVP Build Roadmap, the Payment/Transaction State Model reference, and the Payment Entity design reference. All five are folded in here so there's one place to look. If anything here ever conflicts with the master architecture doc (`ASM guide.html`), the architecture doc wins on *design decisions* (schema, flows, sequencing); this guide wins on *how the code implementing those decisions is written*.

---

## Table of Contents

1. [Where We Are Right Now](#1-where-we-are-right-now)
2. [Golden Rules (Non-Negotiable)](#2-golden-rules-non-negotiable)
3. [MVP Definition & Scope](#3-mvp-definition--scope)
4. [Build Sequence](#4-build-sequence)
5. [Money & Payment/Trip/Settlement State Model](#5-money--paymenttripsettlement-state-model)
6. [Payment Entity & the Grouped-Payment Schema](#6-payment-entity--the-grouped-payment-schema)
7. [Transport Credentials (NFC Cards)](#7-transport-credentials-nfc-cards)
8. [Tap / Authorization Flow](#8-tap--authorization-flow)
9. [Driver Sessions, Fare, Routes, Terminals](#9-driver-sessions-fare-routes-terminals)
10. [Backend Code Conventions (Go)](#10-backend-code-conventions-go)
11. [Database & Schema Conventions](#11-database--schema-conventions)
12. [Frontend Conventions (Flutter + React)](#12-frontend-conventions-flutter--react)
13. [Auth & Permissions (deferred, but designed)](#13-auth--permissions-deferred-but-designed)
14. [Operational Concerns](#14-operational-concerns)
15. [Testing Strategy](#15-testing-strategy)
16. [Offline Resilience, Card Ops Accountability, HCE](#16-offline-resilience-card-ops-accountability-hce-future-path)
17. [Definition of Done](#17-definition-of-done)
18. [PR Checklist & Onboarding](#18-pr-checklist--onboarding)
19. [Immediate Next Task](#19-immediate-next-task)

---

## 1. Where We Are Right Now

**Stack:** Go + Gin + PostgreSQL backend, Moniepoint as payment provider, pilot corridor Tanke ↔ Challenge in Ilorin (Keke first, bus later).

**Implemented foundation:**
- Go + Gin API, PostgreSQL repositories
- User records, passenger linked bank accounts, driver-bound terminals, trip persistence
- Trip financial fields as kobo in `BIGINT`
- Basic endpoints: `/register`, `/link-account`, `/tap`, `/trips`, `/terminals`
- Payment model, payment repository contract + Postgres implementation
- `payments` table with idempotency constraints

**Migrations:**

| # | File | Status |
|---|---|---|
| 0001 | `0001_init.sql` | Applied |
| 0002 | `0002_money_representation.sql` | Applied |
| 0003 | `0003_payment_idempotency.sql` | Applied |
| 0004 | `0004_payer_travellers_and_grouped_payments.sql` | Drafted — introduces `payer_travellers`, `payment_sessions`, turns `payments` into attempts, adds `payment_allocations` |

**⚠️ Two things worth reconciling before going further:**

1. **Grouped payments vs. the roadmap.** Section 4 still frames the system at stage **#5 — Idempotency**, on a simple one-payment-per-trip model ("`trip_id` is not unique because one trip can have multiple payment attempts"). Migration `0004` has already moved past that into a **grouped, multi-traveller payment session** design (one payer, many travellers, one checkout). That concept doesn't appear anywhere in the MVP scope (Section 3) or the #1–25 sequence (Section 4) — worth an explicit call on whether grouped payment is now in MVP scope or `0004` got ahead of the plan.
2. **Money representation status.** The roadmap marks **#3 (money representation) as DONE**, using `BIGINT` kobo fields (`fare_amount`, `passenger_platform_fee`, `driver_platform_fee`, `passenger_total`, `driver_payout`, `platform_total`). But the last confirmed schema check (`\d trips`, run ahead of writing `0003`) showed the `trips` table still on the *old* `numeric(10,2)` columns (`fare_amount`, `driver_payout`, `platform_fee`) — the bigint-kobo breakdown hadn't landed on `trips` itself. Worth a fresh `\d trips` to confirm #3 is actually complete on that table specifically, not just as a convention for newer tables.

**Known current limitation:** `/tap` is still a prototype. A successful response today means the trip/database write succeeded — it does **not** yet mean Moniepoint actually authorized money.

---

## 2. Golden Rules (Non-Negotiable)

Hard rules, not style preferences. If you're unsure whether something violates one, ask before you PR it.

1. **Never touch `*sql.DB` directly in a handler.** Always go through a repository interface — it's the DI seam that keeps the system testable.
2. **Never store mutable state in a package-level variable.** (The original prototype's `var Users = []User{}` was a live, unlocked data race, and it's also what makes horizontal scaling — Section 14.7 — impossible.) State belongs in Postgres or nowhere.
3. **Every query is parameterized.** `$1, $2...` — never string concatenation or `fmt.Sprintf` into SQL.
4. **Secrets never enter source control.** Environment variables only. Pasting a key into a file is a failure even if deleted before commit — rotate it.
5. **Payment and session data is never cached.** Fare tables and dashboard aggregates can be; balances, sessions, and trip state cannot.
6. **Every external call (Moniepoint, VSP, NIMC) has a timeout.** No exceptions.

---

## 3. MVP Definition & Scope

### 3.1 What the MVP is

A real, controlled transport-payment system that completes:

```
Passenger → verified identity → active NFC card → driver terminal →
active driver session → server-resolved fare → payment → Moniepoint
authorization → trip recorded → driver payout/settlement → transaction history
```

Success = a pilot driver can run a full day cash-free, and every successful trip has a clean, auditable financial record.

### 3.2 What the MVP is NOT

Multi-city mobility OS · public partner API · full investor dashboard · BRT integration · USSD product · custom hardware · multi-state expansion · advanced route optimization. All later phases.

### 3.3 What We Do NOT Build Yet

React investor dashboard · Deck.gl investor mode · USSD · BRT integration · taxi partner API · multi-state/multi-city expansion · Redis · horizontal API scaling · advanced PostGIS routing · OSRM route planning · custom hardware · full public partner API.

### 3.4 MVP Pilot Surface

- **Passenger:** enrollment, linked bank accounts, physical NFC card, view fare, tap/pay, transaction history, block lost card, replacement workflow.
- **Driver:** login, start operating session, terminal status, fare display, tap processing, transaction history, receiving account, daily ticket visibility.
- **Field agent:** authenticated agent mode, passenger enrollment, card issuance, lost-card reissue, customer support.
- **HQ/Ops (MVP-level):** driver/session status, terminal status, trip/payment visibility, provisioning/rebinding, reconciliation visibility. Full investor dashboard not required.

### 3.5 MVP Hardware

Budget Android devices (NFC, fingerprint/local auth where required, GPS, connectivity) run the Flutter driver terminal app. PAX A920 Pro is a later fleet transition, not an MVP prerequisite. Custom hardware is later still.

### 3.6 Success Criteria

1. Passenger enrolled
2. Passenger links a payment account
3. Passenger receives an active NFC card
4. Driver has a bound terminal
5. Driver starts a valid operating session
6. System resolves the fare
7. Passenger taps
8. Payment attempt is idempotent
9. Moniepoint authorizes payment
10. Unknown outcomes are reconciled safely
11. Trip is recorded correctly
12. Driver payout is resolved correctly
13. Daily ticket is charged once when applicable
14. Passenger sees transaction history
15. Driver sees fare + ticket history
16. Every sensitive action is auditable
17. Duplicate taps cannot double-charge
18. Tests cover the failure paths
19. Logs and monitoring work
20. The pilot can operate cash-free for a full day

---

## 4. Build Sequence

Deliberately separated so payment/transport correctness comes before auth:

```
FOUNDATION FIXES → REAL PAYMENT FLOW → TRANSPORT OPERATIONS →
AUTHENTICATION → AUTHORIZATION → MVP PILOT
```

| Stage | Status |
|---|---|
| #3 Money representation | DONE (see reconciliation note, Section 1) |
| #4 Payment/transaction state model | DONE |
| #5 Idempotency | IN PROGRESS |
| #6 Payment provider boundary | — |
| #7 Moniepoint integration | — |
| #8 Payment failure / unknown / reconciliation | — |
| #9 Server-controlled fare | — |
| #10 DB transaction / integrity strategy | — |
| #11 NFC passenger card / credential | — |
| #12 Card lifecycle | — |
| #13 Driver operating session | — |
| #14 Fare rules | — |
| #15 Route model | — |
| #16 Driver receiving account | — |
| #17 Daily ticket | — |
| #18 Settlement | — |
| #19 Payment / trip history | — |
| #20 Versioned fee configuration | — |
| #21 Audit logs | — |
| #22 API error system | — |
| #23 Rate limiting | — |
| #24 Observability / logging | — |
| #25 Tests + security hardening + deployment readiness | — |
| *then* #1 Authentication | — |
| *then* #2 Authorization | — |

*Rule: don't implement a later layer just because its architecture is already designed.*

### Stage detail

**#6 Payment provider boundary** — internal `PaymentProvider` interface; the business layer must not know Moniepoint HTTP details. Confirm Moniepoint's real auth, payment operation, account representation, provider references, success/failure/pending outcomes, idempotency mechanism, webhook behavior, timeout behavior, and reconciliation/status lookup before implementing — don't invent the contract.

**#7 Moniepoint integration** — sandbox creds separate from production; store provider references; map provider outcomes to internal states; translate provider errors into stable internal errors; treat timeouts as unknown where appropriate; retries use the correct idempotency key; secrets never in source or logs.

**#8 Payment failure / unknown / reconciliation** — required paths: SUCCESS, FAILURE, TIMEOUT/UNKNOWN, DATABASE FAILURE, PROVIDER SUCCESS + LOCAL FAILURE. Never auto-fallback to another account after an unknown result — safe fallback is limited to explicitly-safe outcomes like confirmed insufficient funds. `Payment=UNKNOWN, Trip=PENDING` → reconcile via provider reference/status.

**#9 Server-controlled fare** — stop trusting client-supplied `amount_kobo`. Target: `terminal → vehicle/route context → fare service → FareQuote → payment authorization`. Driver never types a fare; passenger sees it before confirming.

**#10 DB transaction & integrity strategy** — the tap flow needs real transactions (`BEGIN → validate → write payment → write trip → COMMIT`, rollback on failure). Must prevent: duplicate successful payment for one boarding event, inconsistent payment/trip state, duplicate daily ticket, duplicate settlement, account-selection races, invalid references. External payment calls aren't rolled back by Postgres — reconciliation is still required.

  **Cross-cutting from here on: concurrency safety.** Anything involving money, identity, credentials, state transitions, limits, or settlement must be designed and tested for concurrent execution: payment idempotency/duplicate requests, concurrent payment attempts, account-linking limits, successful-payment-per-trip invariant, daily-ticket charging, card state transitions, driver sessions, settlement/payout, concurrent trip creation, config/version consistency. Principle: *if two requests can arrive at the same time, the system must still produce one correct, deterministic outcome.*

**#11 NFC passenger card / credential** — the card is a credential, not a bank account/wallet/funds source/payout destination. `NFC card → passenger identity → linked bank accounts`. Identifies the passenger without exposing financial credentials. (Full lifecycle in Section 7.)

**#12 Card lifecycle** — see Section 7.

**#13 Driver operating session** — terminal registration ≠ driver session. Needs terminal binding + active driver session + session verification. Session carries driver identity, terminal, start time, expiry, verification method, status. Expired session must not authorize new trips.

**#14 Fare rules** — sources: `explicit_rule`, `route_tier`, `distance_band`. MVP can use a simple single-corridor fare, but it must come from trusted server config, never a driver-entered amount.

**#15 Route model** — MVP starts with one corridor (Tanke ↔ Challenge), but the data model leaves room for `route`, `route_fare_tier`, `terminal_location`. Multi-corridor management is T2.

**#16 Driver receiving account** — payout destination is server-resolved, never accepted from `/tap` (`driver_bank_account`, `payout_destination`, `union_bank_account` must never come from the request). Driver-controlled: `terminal → bound driver → driver receiving account`. Driver needs a verified linked receiving account before payout.

**#17 Daily driver ticket** — separate financial event, one per driver per operating date (`UNIQUE(driver_id, operating_date)`). Trigger: first successful passenger fare of the day → check ticket → if unpaid, debit driver receiving account → pay union account. Ticket failure must not turn a successful fare into a failed one; unknown ticket outcome must never cause an automatic duplicate debit.

**#18 Settlement** — `payment`, `trip`, `settlement` are separate. `Passenger payment → successful collection → settlement record → driver payout.` Settlement has its own state + provider reference.

**#19 Payment/trip history** — Passenger sees: trip, fare, passenger platform fee, passenger total, payment status, provider reference (where appropriate), date/time. Driver sees: fare earned, driver platform fee, driver payout, daily ticket + status, date/time. Never expose full bank account numbers.

**#20 Versioned fee configuration** — don't hardcode fees permanently. Needs effective dates, versioning, privileged changes, audit trail, historical accuracy. Old trips keep their old fee; historical transactions are never silently recalculated.

**#21 Audit logs** — at minimum: card issued/blocked/voided/reissued, terminal bound/rebound, driver session actions, fee changes, daily ticket events, reconciliation actions, admin actions. Each record captures who / what / when / entity / old state / new state / reason.

**#22 API error system** — stable codes instead of ad-hoc strings: `INVALID_REQUEST`, `NOT_FOUND`, `UNAUTHORIZED`, `FORBIDDEN`, `CARD_INACTIVE`, `CARD_BLOCKED`, `SESSION_EXPIRED`, `PAYMENT_FAILED`, `PAYMENT_UNKNOWN`, `DUPLICATE_REQUEST`, `INSUFFICIENT_FUNDS`, `RATE_LIMITED`, `INTERNAL_ERROR`.

**#23–25** — see Section 14 (Operational Concerns) and Section 15 (Testing).

---

## 5. Money & Payment/Trip/Settlement State Model

### 5.1 Money representation — DONE (see Section 1 reconciliation note)

All monetary values are integer kobo in `BIGINT` — never floating point. E.g. ₦510.00 = 51000 kobo.

Trip financial fields: `fare_amount`, `passenger_platform_fee`, `driver_platform_fee`, `passenger_total`, `driver_payout`, `platform_total`.

Example: Fare ₦500 + passenger platform fee ₦10 = passenger pays ₦510. Driver platform fee ₦10 → driver receives ₦490. Platform revenue ₦20.

### 5.2 Why three separate lifecycles

The old `trips.status = "success"` conflated three different questions: did the passenger's payment succeed, did the trip happen, and was the driver/platform actually paid. These are modeled independently:

- **`trips`** → what transport event happened
- **`payments`** → what happened to the passenger's money
- **`settlements`** → what happened to the money owed to recipients

A trip can be `COMPLETED` while settlement is still `PENDING` — that's a valid, expected state, not a bug.

### 5.3 Payment status

| State | Meaning |
|---|---|
| `PENDING` | Record exists, processing not finished |
| `PROCESSING` | Sent to the payment provider |
| `SUCCEEDED` | Provider confirms the passenger was charged |
| `FAILED` | Provider definitively says the charge failed |
| `UNKNOWN` | Provider outcome uncertain — needs reconciliation |

**`UNKNOWN` is never `FAILED`.** If the passenger might already have been charged, an `UNKNOWN` payment must not fall through to another linked account.

### 5.4 Trip status

`PENDING` → `COMPLETED` / `CANCELLED`. (Can later gain journey states like `IN_PROGRESS`/`ENDED` once real journey tracking exists.) For the current tap model, a confirmed successful payment is what allows `COMPLETED`.

### 5.5 Settlement status

Same five states as payment (`PENDING`/`PROCESSING`/`SUCCEEDED`/`FAILED`/`UNKNOWN`), but a distinct lifecycle from payment success.

### 5.6 Core flow

```
Tap → create payment → PENDING → PROCESSING → {FAILED | UNKNOWN | SUCCEEDED}
SUCCEEDED → Trip COMPLETED → Settlement PENDING → PROCESSING →
{FAILED | UNKNOWN | SUCCEEDED}
```

### 5.7 Failure scenarios

- **Payment fails:** `Payment=FAILED, Trip=CANCELLED, Settlement=NOT_DUE`. No driver payout, no platform revenue.
- **Payment succeeds:** `Payment=SUCCEEDED, Trip=COMPLETED, Settlement=PENDING`.
- **Provider times out:** `Payment=UNKNOWN, Trip=PENDING, Settlement=NOT_DUE`. Do *not* auto-retry the next account — the first attempt may have actually succeeded. Handled through idempotency + reconciliation.
- **Provider succeeds, DB write fails** (Moniepoint → SUCCESS, Postgres → ERROR): never classify as failed — the passenger may already be charged. This is why every payment needs a durable record, provider reference, idempotency strategy, and reconciliation.

### 5.8 Core invariant

> **Only `payment = SUCCEEDED` can make a fare financially successful.**

None of these count as proof of success on their own: HTTP 200, trip created, payment request sent, provider request accepted, no immediate provider error.

### 5.9 Reconciliation ≠ payment outcome

Keep them separate fields — `payment_status = SUCCEEDED` + `reconciliation_status = RECONCILED` is clearer than collapsing into one field. Future reconciliation states: `NOT_REQUIRED`, `PENDING`, `RECONCILED`, `DISPUTED`. (Full reconciliation behavior is stage #8.)

---

## 6. Payment Entity & the Grouped-Payment Schema

### 6.1 Design history — why `passenger_id` stays on `payment`

Strict normalization would drop `passenger_id` from `payment` (it's derivable via `trip_id → trips.passenger_id`). **Decision: keep it anyway** — payment history is a major user-facing feature, and keeping the passenger attached directly avoids a join for `SELECT * FROM payments WHERE passenger_id = $1`. Trade-off accepted: the invariant `payments.passenger_id = trips.passenger_id` is maintained by app discipline, not a DB constraint.

Original approved `payment` entity (pre-`0004`):

```
payment
├── id
├── trip_id
├── passenger_id       ← kept despite trip_id redundancy, see above
├── account_id
├── amount
├── status
├── idempotency_key
├── provider
├── provider_reference
├── created_at
└── updated_at
```

Scope note: no `driver_id`, `vehicle_plate`, `bank_name`, `fare`, etc. on `payment` — those live on their own entities. Design lives in two places kept in sync: the migration defines the columns; `internal/models/payment.go` mirrors the row as a `Payment` struct. Database constraints drive the model, not the other way around.

### 6.2 Current schema (as of migration `0004`)

This is the **grouped payment** model, which goes beyond 6.1 and beyond the current roadmap/MVP scope (see the reconciliation flag in Section 1):

- **`payer_travellers`** — who a payer is authorized to pay for (`payer_id`, `traveller_id`, `status: ACTIVE|INACTIVE`), `payer_id ≠ traveller_id`, unique per pair.
- **`payment_sessions`** — one "checkout": `payer_id`, `total_amount`, status (`PENDING|PROCESSING|SUCCEEDED|FAILED|UNKNOWN`), unique `idempotency_key`.
- **`payments`** — now *payment attempts* within a session (`trip_id`/`passenger_id` dropped, `payment_session_id` added). A partial unique index enforces one `SUCCEEDED` payment per session.
- **`payment_allocations`** — maps the one successful payment attempt across the trips it covers (`payment_id`, `trip_id`, `amount`), with `UNIQUE(trip_id)` so a trip can only ever be paid once.

Open items on this schema: confirm `users(id)` is still the correct table name (vs. a possible `identity` rename, see Section 11); no DB-level check yet that `SUM(payment_allocations.amount)` matches the session/payment amount — currently an app-level invariant only.

---

## 7. Transport Credentials (NFC Cards)

### 7.1 What the card is

The company-issued physical NFC card is a credential belonging to a passenger identity — not a bank account, not a wallet, not a funds source, not a payout destination. `NFC card → passenger identity → linked bank accounts`.

### 7.2 States

| State | Meaning |
|---|---|
| `ACTIVE` | May authorize a tap |
| `INACTIVE` | Temporarily disabled by the passenger |
| `BLOCKED` | Security action for lost/stolen/compromised — cannot be reactivated normally |
| `VOID` | Permanently retired, normally after replacement |

Card state is independent of bank-account state and passenger identity state.

### 7.3 State machine

```
ACTIVE ⇄ INACTIVE (passenger toggles)
ACTIVE/INACTIVE → BLOCKED (lost/stolen) → VOID (after replacement) → new card ACTIVE
```

**Invariant:** under the normal physical-card model, there should normally be at most one `ACTIVE` credential per passenger at a time. Historical `VOID` credentials are kept, never deleted.

### 7.4 Activation API

```
GET  /cards/:id
POST /cards/:id/activate     — only valid from the passenger's current INACTIVE credential
POST /cards/:id/deactivate   — reversible by the passenger
POST /cards/:id/block        — immediate, for lost/stolen/compromised
POST /cards/:id/reissue
```

Passenger actions use the authenticated identity from middleware, never a client-supplied passenger ID:

```go
router.POST("/cards/:id/deactivate", RequireRole("passenger"), app.deactivateCardHandler)
router.POST("/cards/:id/block", RequireRole("passenger"), app.blockCardHandler)
```

Backend must verify the credential belongs to the authenticated passenger. A `BLOCKED`/`VOID` card can't be reactivated through the normal endpoint. Card state changes never modify, freeze, or activate any linked bank account. State-changing ops should be idempotent where practical. Every transition produces an auditable event: credential, passenger identity, previous/new state, actor, reason, timestamp.

### 7.5 Authorization rules (pre-payment)

```
Find credential → exists? → resolve identity → state ACTIVE? →
validate terminal → validate driver session → validate fare → resolve payment source
```

`INACTIVE`/`BLOCKED`/`VOID` → reject before payment is attempted.

### 7.6 Reissue

Lost/stolen is deliberately two steps — **block protects immediately; replacement happens after:**

```
Passenger app → authenticate → verify credential belongs to passenger →
BLOCK old card immediately → replacement request/status
```

Passenger can see replacement is pending without the raw card UID being exposed.

Physical replacement:

```
Passenger → identity verification → old credential located →
confirm old credential BLOCKED/eligible → new card UID assigned →
old credential → VOID → new credential created (same identity) →
new credential → ACTIVE → issuance + replacement event recorded
```

Must be atomic from the app's perspective — never two active physical cards for one passenger. If issuance fails, preserve the blocked state; don't silently activate a partial replacement. Never delete the old credential.

### 7.7 Card issuance audit

Eventually record: `credential_id`, `identity_id`, `issued_by`, `issued_at`, `reason`, `previous_credential_id`, `previous_status`, `new_status`, `blocked_reason`, `replacement_request_id`. For agent ops, also: `agent_id`, `location`, `batch_id`.

---

## 8. Tap / Authorization Flow

### 8.1 Target `/tap` orchestration

```
Parse/validate request → authenticate terminal/device → resolve credential →
validate credential state → validate passenger identity → validate terminal →
validate terminal active state → validate driver session → resolve fare →
resolve passenger linked accounts → resolve driver/company receiving account →
attempt payment in priority order → apply platform fee → persist trip →
return transaction result
```

Current `/tap` is a prototype — see the "Known current limitation" note in Section 1.

### 8.2 Passenger identity, linked accounts, and fallback

A passenger identity is separate from financial accounts. A passenger may have **up to three linked bank accounts**, with `priority_order` controlling fallback order. These accounts are **not** a proprietary wallet — the platform never holds passenger funds.

Don't treat `accounts[0]` as the whole production mechanism:

```go
for _, account := range accounts {
    result := payment.Authorize(account, amount)
    if result.Success { return result }
    if result.IsInsufficientFunds() { continue }
    return result.Error
}
```

Only explicitly-safe conditions (e.g. confirmed insufficient funds) proceed to the next account. A timeout or unknown state must never fall through.

### 8.3 Driver/company receiving account

Never accept the payout destination from the tap request.

- Driver-controlled: `terminal → bound driver → driver receiving account`
- Company-controlled: `terminal → operator/company → company receiving account`

This split protects the Keke/bus driver model from being corrupted when BRT/operator-controlled modes arrive later.

### 8.4 Payment processor interface

```go
type PaymentProcessor interface {
    Authorize(
        ctx context.Context,
        source PaymentSource,
        destination ReceivingAccount,
        amount Money,
        idempotencyKey string,
    ) (PaymentResult, error)
}
```

Enables sandboxing, provider swaps, unit testing, and failure simulation.

### 8.5 Idempotency vs. nonce replay

Two different problems, both required:

- **Nonce replay** — prevents the exact signed request from being processed twice.
- **Payment idempotency** — prevents a retried provider call from creating two transfers.

The tap nonce should eventually double as the stable idempotency key for that authorization attempt.

### 8.6 Duplicate tap protection

A second tap can genuinely differ while representing the same boarding event, so use **both**: nonce replay check + a short-window passenger+terminal duplicate check. A recent successful matching trip should be *returned*, not re-charged.

### 8.7 Transaction boundaries

A Postgres rollback cannot undo an external bank transfer. Production handling must explicitly cover: provider timeout, unknown provider result, webhook confirmation, idempotent retry, reconciliation, successful payment followed by DB failure, and successful DB write after provider uncertainty.

**Manual payout retries reuse the original idempotency key, never a fresh transfer.** If ops resends a payout because a driver says they didn't receive it, it calls `moniepoint.Transfer` with the *original trip's nonce*, exactly like the automated path — a fresh ad-hoc transfer with no idempotency key is a real (not hypothetical) way for a driver to get paid twice.

---

## 9. Driver Sessions, Fare, Routes, Terminals

**Driver sessions** — terminal registration ≠ an indefinitely valid session. Needs terminal binding + active driver session + session verification. Session carries driver identity, terminal, route, start time, expiry, verification method, status.

**Fare resolution** — determined before authorization: `route → destination → GetFare → FareQuote → AuthorizeTrip(quote)`. Sources: `explicit_rule`, `route_tier`, `distance_band`. The trip retains its fare source so historical transactions stay explainable after future fare changes.

**Terminals** — a terminal is a physical NFC reader; may collect NFC credential data, GPS, connectivity state, device identity, signed request data. Must **not** contain editable payout config, arbitrary destination accounts, authoritative fee logic, or a secret permitting arbitrary fund routing. Current fields: `terminal_type = nfc_reader`, `vehicle_type = keke | bus` — kept separate. Train is intentionally deferred (operational model is company/operator-controlled).

**Naming trap — `terminal` vs. `terminal_location`:** genuinely different things, easy to reintroduce confusion even after it's fixed in docs. `terminal` = a driver's physical device (`hardware_serial`, bound to one driver). `terminal_location` = a place (Tanke, Challenge). If a variable/param/column would be bare `terminal`/`terminalID`, stop and name it `deviceTerminalID` or `terminalLocationID` explicitly.

---

## 10. Backend Code Conventions (Go)

**Backend authority:** Go + Gin owns authentication, authorization, identity resolution, credential validation, terminal validation, fare authorization, payment orchestration, payout resolution, trip persistence, audit events. Flutter and React are clients — they must not become alternate business-logic engines. Current repositories cover users, passenger accounts, terminals, trips.

**Conceptual request flow:**

```
Handler → Service / use case → Repository interface → PostgreSQL implementation
```

**Package structure** (established, don't deviate):

```
cmd/api/main.go        # package main — wiring only, no business logic
internal/models/       # plain structs, minimal helpers
internal/db/           # connection setup, WithTx helper
internal/repository/   # interfaces + Postgres implementations
```

**Handler pattern:**

```go
func (app *App) someHandler(c *gin.Context) {
    var input struct {
        Field string `json:"field" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    result, err := app.SomeRepo.DoThing(c.Request.Context(), input.Field)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not do thing"})
        return
    }
    c.JSON(http.StatusCreated, result)
}
```

- Input structs defined inline unless reused elsewhere.
- Every handler passes `c.Request.Context()` through to the repository — a dropped client connection then cancels the underlying query.
- Error messages to the client are generic; real detail goes to structured logs only (a verbose error is a reconnaissance gift).

**Repository / `Querier` pattern** — the actual DI seam. Built against an interface satisfied by both `*sql.DB` and `*sql.Tx`, never `*sql.DB` directly:

```go
type Querier interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

This lets the same repository code run standalone or inside `db.WithTx(...)`. Any handler doing more than one write that must succeed/fail together belongs inside `WithTx`.

- Write a fake implementation of any repository interface you add, even before a test exists — if you can't picture the fake, the interface is wrong.
- Don't call `vsp.VerifyFingerprint` or `moniepoint.Transfer` directly from a handler — always through the injected interface (`IdentityVerifier`, `PaymentProcessor`), so tests can substitute a fake.
- Use interfaces for external services generally: `IdentityVerifier`, `PaymentProcessor`, `CardIssuer`, `NotificationService`, `RouteProvider` — for testability and controlled failure handling, not abstraction for its own sake.

**Privileged-mutation pattern** — this shape has independently shown up in `UpdateFare`, `RebindTerminal`, and card batch reconciliation, so it's a real pattern, not a coincidence. Any privileged write that *changes* a value (not just creates a row) uses this shape by default:

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

Old value captured, new value applied, both logged, atomically. Don't invent a fourth variant of this.

---

## 11. Database & Schema Conventions

**Migrations:**

- Numbered sequentially (`0001_init.sql`, `0002_...`), never reordered or renumbered after merge, and never edited after being applied (don't rewrite history).
- One migration = one logical change; don't bundle an unrelated tweak into a migration named for something else.
- Additive by default (`ADD COLUMN IF NOT EXISTS`). Destructive migrations (drop column/table) need a second reviewer and a written note on how existing data is handled.

**Schema conventions:**

- Primary keys: `UUID`, always.
- Columns: `snake_case`.
- Timestamps: `TIMESTAMPTZ`, never bare `TIMESTAMP` (timezone-naive timestamps are a bug waiting to surface across regions).
- Foreign keys: `<referenced_table_singular>_id` — `driver_identity_id`, not `driver_id_fk` or `fk_driver`.
- Booleans: `is_active`, `is_verified` — prefixed, never bare `active`.

**PostGIS:**

- Geometry columns: `geometry(Point, 4326)` / `geometry(LineString, 4326)` — 4326 (WGS84) always, don't mix SRIDs.
- Spatial queries use `ST_DWithin` with `::geography` casts for real-world distance — raw `ST_Distance` on `geometry` gives degrees, not meters, and it's a subtle bug that only surfaces when someone eyeballs the numbers.

(See Section 9 for the `terminal` vs. `terminal_location` naming trap.)

---

## 12. Frontend Conventions (Flutter + React)

### 12.1 Flutter (Driver / Passenger)

Feature-first folder structure, not layer-first — each feature owns its own `screens/`, `widgets/`, `providers/`:

```
lib/
├── core/           # config, constants, network client, security (device-bound key)
├── data/           # models, repositories, datasources (remote + local)
├── features/
│   ├── auth/
│   ├── driver_home/
│   │   ├── session/ nfc_tap/ fallback/ gis_tracking/ bank_details/
│   │   ├── transaction_history/
│   │   └── vehicle_mode/keke/
│   ├── passenger_home/
│   │   linked_accounts/
│   │   card/ (status, activate, deactivate, block, replacement_status)
│   │   trip_history/
│   ├── admin_home/       # mobile, monitoring-only — never build privileged actions here
│   ├── ops/               # staff-only, HQ
│   └── agent_ops/         # staff-only, field
├── presentation/
└── state/
```

Flutter has not yet been implemented; this is the target shape for when it starts. HCE is later scope; driver functionality stays separate from passenger.

**Naming:** Screens `PascalCase` ending `Screen` (`ShiftStartScreen`, not `shift_start_page`); Riverpod providers end `Provider` (`sessionStateProvider`); repositories end `Repository`, matching Go-side naming so a dev can find the corresponding backend interface without guessing.

**New feature checklist:** create the feature folder → define the repository interface in `data/repositories/` before any UI (mirrors Go-side discipline) → wire a Riverpod provider for state outliving one widget → build the screen last, against the provider (never call the repository directly from a widget).

**Do:** cache route/destination lists client-side for the shift; use `flutter_secure_storage` for the device-bound key or cached biometric template (never `SharedPreferences`).
**Don't:** put business logic (fare calc, session validity) in a widget; call `dio` directly from a screen — every network call goes through `core/network`'s signing interceptor.

### 12.2 React (Admin dashboard)

For live map, provisioning, driver oversight, passenger/credential oversight, card issuance/reconciliation, route management, fare management, financial reconciliation, audit review. Privileged mutations stay authorized by Go.

**TypeScript mandatory**, not a preference. Interfaces mirror Go structs field-for-field:

```typescript
// Go: type Session struct { ID string; DriverID string; ExpiresAt time.Time; Status string }
interface Session {
  id: string;
  driverId: string;
  expiresAt: string; // ISO 8601 — Go's time.Time marshals to this
  status: 'active' | 'expired' | 'flagged';
}
```

If a Go struct changes, the TS interface changes in the *same* PR, not a follow-up.

**Do:** keep privileged actions (rebind, reconciliation, fare edits) behind explicit confirmation dialogs — they hit `UpdateFare`/`RebindTerminal`/etc. and log an admin ID; never fire on a single click.
**Don't:** fetch dashboard aggregates (heatmap, corridor stats) on every render — respect the caching layer's short TTLs.

---

## 13. Auth & Permissions (deferred, but designed)

Deliberately delayed until #25 (Section 4) — the system underneath needs to be coherent first. Almost none of this is in the real codebase yet; it's still on the stopgap terminal-key check from the `/tap` fix. This is the spec to build against once that point is reached.

**Logins needed:** Passenger, Driver, Staff/agent, Admin. Authenticated identity must come from the server-side session/token — never trust `passenger_id`/`driver_id`/`admin_id` supplied by the client for authorization decisions.

**Middleware, not inline checks** — `if session.Role != "driver"` inside a handler body gets copy-pasted inconsistently as routes grow:

```go
router.POST("/tap", RequireRole("driver"), app.tapHandler)
router.POST("/admin/rebind", RequireRole("admin"), RequireAudit(), app.rebindHandler)
```

**Admin RBAC is `ops` vs. `super_admin`** — don't collapse into a single "is admin" boolean. Rebind and card-batch reconciliation need `super_admin`; read-only dashboard views need only `ops`. Field: `identity.admin_tier`, checked the same middleware way:

```go
router.POST("/admin/rebind", RequireAdminTier("super_admin"), RequireAudit(), app.rebindHandler)
```

**No public admin registration endpoint, ever** — not even "temporary" for testing. Only two ways an admin account is created: (1) the first account via a one-time deployment script, not the running app; (2) every account after, via `InviteAdmin`, callable only by an existing `super_admin`, itself logged/audited like any other privileged action.

**WebAuthn/passkeys** — use a real library (`go-webauthn` or equivalent), never hand-roll the challenge-response verification; replay and origin-binding edge cases are exactly the kind of security-critical code that gets a subtle vuln shipped when hand-rolled.

**Session tokens are never logged**, not even at debug level — log the session *ID*, never the token.

**Authorization examples once auth exists:**

```
Passenger   → own cards, own accounts, own trips
Driver      → own operating session, own terminal, own trips, own receiving account
Agent       → passenger enrollment, card issuance/reissue within assigned scope
Ops         → operational management
Super Admin → privileged configuration and oversight
```

---

## 14. Operational Concerns

### 14.1 Hosting & Deployment

Multi-stage Dockerfile — build stage compiles the Go binary, final stage is a minimal image with just the binary; don't ship the Go toolchain to production. Deployment checklist every time: migrations run *before* the new binary deploys (never after) → `/health` returns 200 before traffic routes to the new instance → old instance drains (graceful shutdown, 15s window) before being killed.

### 14.2 Environments & Secrets

Three environments minimum: dev, staging, production. Never test against production data — staging uses Moniepoint sandbox credentials, never live (mixing these up is how real money moves during a test). Secrets are environment-scoped; a staging `DATABASE_URL` and a production one are different values in different places, never reused "to save time." Env vars only, `SCREAMING_SNAKE_CASE`, prefixed by concern where helpful (`DATABASE_URL`, `MONIEPOINT_API_KEY`, `VSP_ENDPOINT`). Never hardcode DB passwords, payment keys, identity-provider credentials, webhook secrets, or signing secrets — dev can use a local env var, production needs proper secret management.

### 14.3 CI/CD & Version Control

Trunk-based branching — short-lived feature branches off `main`, merged via PR, no long-running `develop` accumulating drift. Commit messages describe *why*, not just *what* ("Add replay protection to /tap", not "update main.go"). CI pipeline: lint/vet → build → test → (on merge to `main`) build Docker image → deploy. Every PR must: pass CI (`go vet`, `go build ./...`, tests, secret scan); include a test for any change to session/auth/payment logic, no exceptions; note in the description if it touches a Golden Rule area.

### 14.4 Security & Row-Level Security

Redaction happens once, centrally, in logging middleware — not per call site (you'll eventually miss one). RLS (T3+) is defense-in-depth against a bug in your own authorization logic — write the Go-side authorization correctly first; RLS is a safety net, not a substitute. Any PR touching `AuthorizeTrip`, `StartDriverSession`, device binding, or payout resolution needs a second reviewer, always, regardless of how small it looks.

### 14.5 Rate Limiting

New endpoint → new explicit rate limit; don't assume the global default covers it:

```go
router.POST("/tap", RateLimitByTerminal(), app.tapHandler)
```

Per-terminal limits for driver-facing endpoints, per-IP for public ones. MVP targets: `/tap`, `/location`, public endpoints — key `/tap` by terminal/device identity where possible. If an endpoint doesn't fit either pattern, ask before shipping it unlimited.

### 14.6 Caching & CDN

Only cached: dashboard aggregates (heatmap, corridor stats, revenue tickers) with a short TTL (30–60s), and route/stop lists client-side per shift. Never cached: session state, account balances, fare calculations for an in-progress trip, anything from `AuthorizeTrip`. If unsure whether something's safe to cache: no, until someone with full context on the payment flow says otherwise.

### 14.7 Load Balancing & Scaling

Every handler must be stateless (this is Golden Rule #2, restated because it's the direct reason horizontal scaling works). `/ws/map` is the one real exception to plan for: once there's more than one API instance, WebSocket position updates need a shared pub/sub (Redis) to fan out across instances. Don't build this before it's needed (single instance is fine through T3), but don't architect yourself into a corner where it's hard to add.

### 14.8 Logging & Error Tracking

Structured logging always (`log/slog`, key-value fields, never a bare formatted string you'll have to regex later):

```go
slog.Error("trip authorization failed", "terminal_id", terminalID, "error", err)
// not: log.Printf("failed to authorize trip for terminal %s: %v", terminalID, err)
```

Safe identifiers to log: `trip_id`, `payment_id`, `terminal_id`, `session_id`, `credential_id`. **Never log, at any level:** session tokens, payment secrets/API keys, full account numbers, private signing keys, raw biometric data (fingerprint template hashes included). If a struct might contain one of these, log specific safe fields instead of the whole struct.

Sentry (or equivalent) catches unhandled errors — it's not a substitute for handling errors you already know how to handle; an `if err != nil` you're unsure about is a code smell, not a Sentry ticket.

### 14.9 Availability & Recovery

Every external call gets a timeout, no exceptions:

```go
ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
defer cancel()
```

`/health` checks real dependencies, not just "the process is running":

```go
router.GET("/health", func(c *gin.Context) {
    if err := app.Conn.PingContext(c.Request.Context()); err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unreachable"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
})
```

Backups are *verified*, not just scheduled — a silently-failing cron job isn't a backup; confirm a restore actually works. (See Section 8.7 for the manual-payout-retry idempotency rule.)

---

## 15. Testing Strategy

**Unit tests:** card state transitions + audit events, account priority selection, fare logic, middleware, duplicate-tap rules, nonce handling, payout resolution, session expiration, idempotency, daily-ticket uniqueness, error mapping.

**Integration tests:** `tap → credential → session → account selection → payment → trip → settlement`, using mocked external payment/identity providers.

**Failure tests — explicitly cover:** insufficient funds, provider rejection, provider timeout, unknown payment, duplicate idempotency key, duplicate tap, inactive/blocked/void/unknown card, duplicate active-card creation, invalid card transition, inactive terminal, expired session, database failure, payment/database inconsistency, duplicate daily ticket.

**Security/deployment readiness (before pilot):** no secrets in git; separate sandbox/production credentials; least-privilege Postgres user; parameterized SQL; TLS; dependency vulnerability scanning; nightly DB backups; `/health`; uptime monitoring; graceful shutdown; CI build/vet/test; Docker deployment; structured/redacted logs.

---

## 16. Offline Resilience, Card Ops Accountability, HCE Future Path

**Offline resilience** — design it in, don't bolt it on. A client may eventually queue signed requests, but the backend must still enforce replay, stale-request, duplicate-submission, and terminal-compromise rules regardless.

**Card operations & accountability** — eventual flow: `card inventory → batch allocation → agent issuance → passenger credential → activation → deactivation → block → reissue → reconciliation`. Agents must not be able to silently create unlimited cards; card stock must be accountable.

**HCE future path** — a future passenger app may emulate the transport credential on Android. Preferred model: `reader challenge → phone private key → signed response → backend verification`. Never make a static identifier the security mechanism. Physical cards remain the universal fallback. iOS is deferred to Apple's Express Transit entitlement.

---

## 17. Definition of Done

A feature isn't done because the happy path works. Ask:

- Can it authenticate the caller? Can it reject unauthorized access?
- Can it be retried safely? Can it fail halfway? Can it recover?
- Can it be audited? Can it be tested without external services?
- Can the database remain consistent? Can the user understand the result?
- Can an operator investigate it later?

**For payment features, additionally:**

- Can it charge twice? Can money move to the wrong destination?
- What happens when the provider times out? When the response is lost? When the database fails?
- What happens when the card is blocked immediately before a tap?
- Can a replacement ever leave two active cards? Is the old credential retained but permanently unable to authorize?

---

## 18. PR Checklist & Onboarding

**PR checklist (copy into every PR description):**

- [ ] Follows folder/naming conventions for its layer
- [ ] No new package-level mutable state
- [ ] All queries parameterized
- [ ] Secrets via environment variables only
- [ ] Tests included if touching session, auth, or payment logic
- [ ] Structured logging, no sensitive fields logged
- [ ] External calls have timeouts
- [ ] Second reviewer tagged if touching a Golden Rule area

**New developer onboarding:**

1. Read the master architecture doc (`ASM guide.html`) — schema, pseudocode flows, security fixes and *why* they exist.
2. Read this guide's Golden Rules (Section 2) twice.
3. Set up dev environment against **staging**, never production — confirm `DATABASE_URL` points where you think it does before running anything.
4. First PR should be small and touch exactly one layer — better for learning conventions than reading about them.

---

## 19. Immediate Next Task

```
#5 Idempotency
     ↓
Payment service
     ↓
Duplicate-key behavior
     ↓
Concurrent-request handling
     ↓
Tests
```

Then: `#6 PaymentProvider → #7 Moniepoint → #8 failure/reconciliation`. **Don't jump to Flutter or React until the backend payment path is structurally ready.**

Idempotency-stage payment entity shape:

```
payment
├── id
├── trip_id
├── passenger_id
├── account_id
├── amount
├── status
├── idempotency_key
├── provider
├── provider_reference
├── created_at
└── updated_at
```

Rules: one idempotency key = one payment attempt; retrying the exact same attempt reuses the same key; a fallback attempt against another account gets a new key; `trip_id` is not unique (one trip can have multiple attempts) — *though note the reconciliation flag in Section 1: this assumption is already superseded by migration `0004`'s session model.* The database unique constraint is the final duplicate-protection layer; concurrent requests must still be protected by it.

---

## Final MVP Principle

> The MVP is not the smallest number of features. It is the smallest system that can move real money safely, identify the right passenger and driver, record what happened, recover from uncertainty, and explain every transaction afterwards.

---

*This file supersedes: `ENGINEERING_GUIDE.md` (main), the additional card/payment implementation guide, the MVP Build Roadmap, the Payment/Transaction State Model reference, and the Payment Entity design reference. The additional guide's original §31 "Engineering Sequence" (a coarser 21-item list) is superseded by the detailed #1–25 sequence in Section 4 and was dropped here to avoid two conflicting sequences in one place.*
