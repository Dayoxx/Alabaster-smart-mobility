# Master Engineering Guide: Upgrade v2

Drop-in replacement and addition sections for `MASTER_ENGINEERING_GUIDE.md`. Anything not listed here stays as it is. Written October 2026.

## What this upgrade changes, and why

| # | Area | Problem in the current guide | Fix in this upgrade |
|---|---|---|---|
| 1 | Section 1 (status) | Says migration 0004 is "drafted" and lists no handlers or Moniepoint package. Your repo now has migrations 0001–0005, `internal/handlers/`, `internal/moniepoint/` and the grouped-payment models | New status section with a verify-first checklist |
| 2 | Section 1 (two open flags) | Both reconciliation flags are still unresolved | Turned into numbered decisions (new Section 20) |
| 3 | Section 7 (credentials) | Written only for physical NFC cards | One credential model for card, bracelet and phone |
| 4 | New Section 7A | Bracelets are not covered | Tiers, tap flow, key rules, schema, tests |
| 5 | Section 5.8 vs bracelet plan | The bracelet build guide says "you are the float" and settles later. Your core invariant says only a succeeded payment makes a fare successful. They conflict | Decision D2, with a recommendation |
| 6 | Section 10 (structure and `Querier`) | Package list does not match the repo, and `Querier` is written for `database/sql` while the project is moving to `pgx/v5` | Updated layout and a driver check |
| 7 | Sections 14 and 15 | No bracelet-specific rate limits, logging rules or failure tests | Additions |
| 8 | Section 19 | Next task list is stale | New order of work |
| 9 | Docs folder | Five superseded guides still sit beside the master guide | Housekeeping list |

---

## Replacement for Section 1: Where We Are Right Now

**Stack:** Go + Gin + PostgreSQL, Moniepoint as payment provider, pilot corridor Tanke ↔ Challenge in Ilorin (Keke first, bus later).

**Present in the repository:**
- `cmd/api/` entry point
- `internal/handlers/`: accounts, app, register, tap, terminals, trips
- `internal/models/`: passenger account, payer traveller, payment, payment allocation, payment session, terminals, trip, user
- `internal/repository/`: payer traveller, payment allocation, payment, payment session, postgres, repository
- `internal/services/payment_service.go`
- `internal/moniepoint/` provider package
- `migrations/0001`–`0005`

| # | Migration | State to confirm |
|---|---|---|
| 0001 | `init` | Applied |
| 0002 | `money_representation` | Applied |
| 0003 | `payment_idempotency` | Applied |
| 0004 | `payer_travellers_and_grouped_payments` | In repo. Confirm it is applied |
| 0005 | `trip_financial_invariants` | In repo. Confirm it is applied |

**Verify before trusting this table.** Run these against the live `transportation_model` database and record the results here:

```
\dt                      -- which tables exist
\d trips                 -- are the fare and fee columns BIGINT kobo or still numeric(10,2)?
\d payments              -- attempt model, partial unique index on SUCCEEDED
SELECT * FROM schema_migrations;   -- if a migration tool is in use
```

The earlier guide flagged that `trips` might still hold the old `numeric(10,2)` columns. Migration 0005 suggests this was addressed, but only `\d trips` confirms it.

**Not yet in the repo:** the `identity` rename, the polymorphic audit log table, the `credential` table, route and fare-engine tables, sessions, and the bracelet tables. These are the next migrations (0006 onward).

**Known limitation:** `/tap` is still a prototype. A successful response means the trip and database write succeeded. It does not yet mean Moniepoint authorized money.

---

## Replacement for Section 7: Transport Credentials

### 7.1 What a credential is

A credential is something a passenger presents at a terminal to identify themselves. It is not a bank account, wallet, funds source or payout destination.

```
credential → passenger identity → linked bank accounts
```

Supported types:

| Type | Form | Status |
|---|---|---|
| `card` | Physical NFC card | Current |
| `bracelet_desfire` | Passive NFC bracelet, Standard tier (see 7A) | Replacing the card |
| `bracelet_javacard` | Passive NFC bracelet, Elite tier | After the Standard pilot |
| `phone_hce` | Phone-as-card, Android first | Later |

All types share one state machine, one table (`credential`), one audit trail and one authorization path. The type decides only how the credential proves itself at the tap.

### 7.2 Unified states

The old guide used ACTIVE, INACTIVE, BLOCKED and VOID. The bracelet plan used Active, Off, Blocked and Suspended. These are merged as follows:

| State | Meaning | Who can set it |
|---|---|---|
| `ACTIVE` | May authorize a tap | Passenger, platform |
| `INACTIVE` | Switched off by the passenger (the "Off" state). Reversible | Passenger |
| `SUSPENDED` | Frozen by the platform for fraud or investigation. The passenger cannot lift it | Platform staff only |
| `BLOCKED` | Lost, stolen or compromised. Not reactivated through the normal endpoint | Passenger, platform |
| `VOID` | Permanently retired, normally after replacement | Platform |

Card state is independent of bank-account state and of passenger identity state.

### 7.3 State machine

```
INACTIVE ⇄ ACTIVE            (passenger toggles)
ACTIVE / INACTIVE → SUSPENDED → ACTIVE / BLOCKED   (platform only)
ACTIVE / INACTIVE / SUSPENDED → BLOCKED → VOID → replacement credential ACTIVE
```

A new credential is created `INACTIVE` and shipped that way, so a lost parcel is harmless.

**Invariant:** a passenger has at most one `ACTIVE` physical credential (card or bracelet) at a time. Historical `VOID` credentials are kept and never deleted. Whether a `phone_hce` credential may be active alongside a physical one is decision D4 in Section 20.

Every transition runs in one database transaction that also writes an audit event: credential, identity, previous state, new state, actor, reason, timestamp. This is the privileged-mutation pattern from Section 10.

### 7.4 Activation API

Unchanged in shape. Routes are `/credentials/:id/...` rather than `/cards/:id/...`, so they cover every type. Add `/credentials/:id/suspend` and `/credentials/:id/unsuspend` for platform staff only.

### 7.5 Authorization rules before payment

```
Authenticate the credential (type-specific, see 7A.3) → look up credential →
resolve identity → state is ACTIVE? → validate terminal → validate driver session →
validate fare → resolve payment source
```

Any state other than `ACTIVE` is rejected before a payment is attempted.

### 7.6 Replacement and upgrade

Lost or stolen is two steps. Block protects immediately. Replacement follows.

```
Passenger authenticates → verify the credential belongs to them →
BLOCK old credential immediately → open replacement request
```

Physical replacement and Standard-to-Elite upgrade use the same flow: the new credential is linked to the same identity, the old one is blocked then voided, and unsettled trips move across. History and tier belong to the account, not the device.

Replacement must be atomic. Never two active physical credentials for one passenger. If issuance fails, the old credential stays blocked and no partial replacement becomes active.

### 7.7 Issuance audit

Record: `credential_id`, `identity_id`, `type`, `issued_by`, `issued_at`, `reason`, `previous_credential_id`, `previous_state`, `new_state`, `blocked_reason`, `replacement_request_id`. For agent issuance also `agent_id`, `location`, `batch_id`. For bracelets also `tier`, `key_version`, `provisioning_run_id`.

---

## New Section 7A: Bracelet Credentials

### 7A.1 Scope

A battery-free NFC bracelet replaces the physical card as the main credential. The bracelet identifies the rider at the tap. There is no balance on it. Money still moves through Moniepoint exactly as Section 8 describes.

| Tier | Chip | Keys | Offline verification |
|---|---|---|---|
| Standard | MIFARE DESFire EV3 | AES keys derived on the server from a master key | No, the terminal needs the server |
| Elite | Java Card secure element with your own applet | Private key generated on the chip, never leaves it | Yes, using your certificate authority |

**Baseline protection is the same for every tier:** Off/Active/Block, tap notifications, server-side limits and a dispute button. Elite sells higher limits, stronger assurance, finish and service. Do not paywall basic safety.

### 7A.2 Schema

```
credential   (id, owner_id, type, state, chip_uid, version, created_at)
bracelet     (credential_id PK/FK, tier, edition, key_version, provisioned_at)
```

- `chip_uid` is unique and is never returned to clients or logged.
- `version` is the optimistic-locking column. State changes use `WHERE id = $1 AND version = $2`.
- `key_version` records which master key derived the chip's key, so the master key can rotate without breaking bracelets in riders' hands.
- Add a nullable `credential_id` to `trips`, so disputes and the audit trail can link a ride to the exact credential used.
- Do this in migrations 0007 and 0008, after the `identity` rename and audit table (0006). Follow Section 11 conventions: UUID keys, `TIMESTAMPTZ`, `is_` prefixed booleans.

### 7A.3 Tap flow

A card or phone tap is one request. A DESFire tap needs several rounds between the chip and your server, so the terminal relays chip messages.

```
POST /tap/bracelet/start    terminal ID, nonce, credential type → first chip command
POST /tap/bracelet/step     chip response → next chip command, or "done"
POST /tap/bracelet/finish   server checks state, limits, terminal, then calls payment_service
```

Rules:

1. `finish` must reuse the existing path into `payment_service`. Idempotency, nonce replay protection and the duplicate-tap check apply unchanged.
2. Each in-progress tap is a short-lived server-side session with a timeout of a few seconds. In-memory is acceptable on a single instance. Moving to more than one API instance means moving this state to Redis or Postgres first (Section 14.7).
3. The terminal sends the credential type with each tap. Everything after authentication is shared by all types.
4. Authentication code sits behind one interface:

```go
type Authenticator interface {
    Type() string
    Start(ctx context.Context, tc TapContext) (Session, error)
}

type Session interface {
    Next(ctx context.Context, chipResponse []byte) (command []byte, done bool, err error)
    Credential() (credentialID uuid.UUID, err error)
}
```

Implementations: `desfire`, `javacard` (later), and `hce` (the existing phone path moved behind the same interface). Write a fake implementation first, so `tap_service.go` is testable before the real DESFire code exists.

### 7A.4 Key management (Standard tier)

- One master key, held in an HSM or cloud KMS. For development, a local key file stored outside the repository.
- Each bracelet gets its own AES-128 key derived from the master key and chip data, using NXP's diversification method.
- The master key never reaches a phone, a terminal or a log. It is never committed. If one is pasted anywhere, rotate it.
- Provisioning runs only from `cmd/provision` on a locked-down machine and writes an audit record for every run.
- **Unresolved design question:** the UID that feeds key diversification is hidden once Random ID is enabled. The server then needs another way to know which key to try before authenticating, for example a freely readable app-level ID file. Settle this on spare chips before enabling Random ID on any real chip, because Random ID cannot be undone.

### 7A.5 Known risks

| Risk | Effect | Mitigation |
|---|---|---|
| Relay latency | Relaying every message from a budget Android terminal to the server may exceed the one-second tap target | Measure early. Fall back to per-terminal session keys or a hardware SAM |
| Library support | Go support for DESFire EV2/EV3 secure messaging is uncertain | Check before buying readers. Budget for implementing the secure messaging layer from NXP's documents |
| Rogue reader | A generic reader tries to read or charge a bracelet | Test that it obtains nothing useful |
| Captured tap | A recorded tap is replayed | Server-issued challenge and the nonce replay check |
| Chip certification | Certification of the chip does not cover your Elite applet | Budget an independent security review |
| Regulation | Taking debits and paying drivers can be a licensed activity | Use a licensed provider and get local advice |

---

## Additions to Section 5.8: Core invariant

Keep the invariant exactly as written:

> Only `payment = SUCCEEDED` can make a fare financially successful.

The bracelet changes how a rider is identified. It does not change how money moves. See decision D2 in Section 20 before adding any "approve now, settle later" behavior.

---

## Additions to Section 8: Tap / Authorization Flow

In 8.1, insert a new first step after parsing the request: **authenticate the credential** according to its type (7A.3). The rest of the flow follows from a verified credential.

In 8.5, add: for bracelet taps, the idempotency key is derived from the terminal nonce plus the tap session ID. A retried `finish` call with the same session must return the original result and never create a second payment attempt.

---

## Replacement for Section 10: Package structure and `Querier`

**Package structure (matches the current repo, plus new packages):**

```
cmd/api/main.go            # wiring only
cmd/provision/main.go      # NEW: bracelet provisioning CLI
internal/db/               # pool, WithTx helper
internal/handlers/         # HTTP in and out
internal/models/           # plain structs
internal/moniepoint/       # provider client, behind a PaymentProcessor interface
internal/repository/       # interfaces + Postgres implementations
internal/services/         # business rules (payment_service, tap_service)
internal/credential/       # NEW: Authenticator interface, desfire/, javacard/
```

**Check the database driver.** The `Querier` interface in the current guide is written for `database/sql` (`ExecContext`, `QueryContext`, `QueryRowContext`). If the project has moved to `pgx/v5`, define `Querier` over the pgx equivalents (`Exec`, `Query`, `QueryRow`, satisfied by both `pgxpool.Pool` and `pgx.Tx`), and update the Golden Rule 1 wording from `*sql.DB` to the real pool type. Do this in one PR so the guide and the code agree.

Keep the privileged-mutation pattern unchanged. Credential state changes, key rotation and tier upgrades all use it.

---

## Additions to Section 14: Operational Concerns

**14.5 Rate limiting.** Give the three bracelet endpoints explicit limits keyed by terminal identity. `start` is the one an attacker would flood, so it gets the tightest limit and a short expiry on unfinished sessions.

**14.8 Logging.** In addition to the existing never-log list, never log: chip UIDs, raw chip messages, derived keys, or anything from `cmd/provision` that includes key material. Log `credential_id` and `tap_session_id` instead.

**New 14.10 Hardware readiness.** Before the bracelet pilot: measure read range on a table and on a wrist, test at the angles drivers actually see, flex and sweat tests, and a rogue-reader test. Keep results in the hardware repository, not in this one.

---

## Additions to Section 15: Testing

Add these to the failure tests:

- A captured tap is replayed and rejected
- A generic reader gets nothing useful from a bracelet
- A tap is abandoned between `start` and `finish`, then the session times out with no charge
- Two terminals tap the same bracelet at once and exactly one trip results
- A bracelet is blocked immediately before a tap, and the tap is declined within a couple of seconds
- A `SUSPENDED` credential cannot be reactivated by the passenger
- An upgrade leaves exactly one active physical credential, and unsettled trips move to the new one
- The Authenticator fake can simulate a slow chip, a malformed response and an authentication failure

---

## Replacement for Section 19: Immediate Next Task

**Main track (unchanged order):**

```
#5 Idempotency → payment service → duplicate-key behavior →
concurrent-request handling → tests
```

Then #6 PaymentProvider, #7 Moniepoint, #8 failure and reconciliation. The Flutter and React rule stands: do not start them until the backend payment path is structurally ready.

**Parallel track (does not block the main track):**

1. Confirm migrations 0004 and 0005 are applied (Section 1 checklist)
2. Make decisions D1 and D2 (Section 20)
3. Migration 0006: `identity` rename, audit log table, UUID key switch
4. Migrations 0007 and 0008: `credential` and `bracelet`
5. `internal/credential` with the fake authenticator, then `tap_service.go`
6. Python desk spike with a DESFire sample chip, in a throwaway folder or separate repo
7. `cmd/provision` once the spike has settled the Random ID question

Do not enable bracelet pilots until #5 to #10 are done.

---

## New Section 20: Decision Log

| ID | Decision | Status | Recommendation |
|---|---|---|---|
| D1 | Is grouped payment (one payer, many travellers) part of the MVP? Migration 0004 built it, but Sections 3 and 4 do not mention it | Open | Keep the schema. For the MVP, the only exposed case is payer = traveller, so there is no new screen or flow. Record the full grouped flow as post-MVP in Section 3 |
| D2 | Settlement model: does the platform ever approve a tap before the money is collected ("float")? | Open, and blocks bracelet settlement code | Keep the Section 5.8 invariant for the MVP. The bracelet is only a different credential. Revisit float only with a licensed provider and legal advice |
| D3 | Credential states are unified as in 7.2 | Proposed | Adopt |
| D4 | May a `phone_hce` credential be active alongside a physical one? | Open | Allow it later, as an explicit exception to the one-active-credential rule, with its own audit |
| D5 | Database driver: `database/sql` or `pgx/v5`? | Check | Match the guide to whatever the code uses, see Section 10 |
| D6 | When does the `users` → `identity` rename and the UUID key switch land? | Open | Before `credential` is created, so the new table starts with the right names and key types |

When a decision is made, change its status, add the date and a one-line reason. Do not delete rows.

---

## Docs housekeeping

1. Move the five superseded guides and the old roadmaps into `docs/archive/`: `1 ENGINEERING_GUIDE.md`, `Addittional ENGINEERING_GUIDE.md`, `ALABASTER_MVP_BUILD_ROADMAP.md`, `ASM_MODEL_PAYMENT_TRANSACTION_STATE_MODEL.md`, `payment_entity_reference.md`, `oldroadmap.html`. Keep the files themselves, only move them.
2. Keep at the top level of `docs/`: `MASTER_ENGINEERING_GUIDE.md`, `ASM roadmap.html` and a new `BRACELET.md` (a short build guide, trimmed to what you decided).
3. Remove `tree.txt` from the repository, or add it to `.gitignore`.
4. Add a root `.gitignore` covering env files, key files and provisioning output.
5. Update the "This file replaces" line at the top of the master guide to say the archived files now live in `docs/archive/`.
6. Update the Table of Contents to add 7A and 20.
