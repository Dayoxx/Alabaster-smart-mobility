# Alabaster Smart Mobility — MVP Build Roadmap
## Revised implementation sequence: fix the foundation first

**Status:** Active engineering roadmap  
**Scope:** MVP only  
**Current backend:** Go + Gin + PostgreSQL  
**Payment provider:** Moniepoint  
**Pilot:** Ilorin Keke, initially Tanke ↔ Challenge  
**Primary rule:** Finish the existing-system fixes before building Authentication (#1) and Authorization (#2).

---

## 1. MVP Definition

The MVP is a real, controlled transport-payment system that can complete this flow:

```text
Passenger
   ↓
Verified passenger identity
   ↓
Active NFC transport card
   ↓
Driver terminal
   ↓
Active driver operating session
   ↓
Server-resolved fare
   ↓
Passenger payment
   ↓
Moniepoint authorization
   ↓
Trip recorded
   ↓
Driver payout / settlement
   ↓
Driver transaction history
```

The MVP is successful when a pilot driver can operate a full day cash-free and every successful trip has a clean, auditable financial record.

### MVP is NOT

- A multi-city mobility OS
- A public partner API
- A full investor dashboard
- A BRT integration
- A USSD product
- Custom transport hardware
- Multi-state expansion
- Advanced route optimization

Those remain later phases.

---

# 2. Current Project State

Implemented foundation:

- Go + Gin API
- PostgreSQL connection and repositories
- User records
- Passenger linked bank accounts
- Driver-bound terminals
- Trip persistence
- Trip financial fields represented as kobo using `BIGINT`
- Basic `/register`, `/link-account`, `/tap`, `/trips`, `/terminals`
- Payment model
- Payment repository contract
- PostgreSQL payment repository
- `payments` table and idempotency database constraints

Current database migrations:

```text
0001_init.sql
0002_money_representation.sql
0003_payment_idempotency.sql
```

`0003` is applied successfully.

### Important current limitation

The current `/tap` implementation is still a prototype. A successful response currently means the trip calculation/database path succeeded; it does **not** yet mean that Moniepoint actually authorized money.

---

# 3. Revised Build Order

We are intentionally separating:

```text
FOUNDATION FIXES
      ↓
REAL PAYMENT FLOW
      ↓
TRANSPORT OPERATIONS
      ↓
AUTHENTICATION
      ↓
AUTHORIZATION
      ↓
MVP PILOT
```

The working order is:

```text
#3  Money representation                         DONE
#4  Payment / transaction state model             DONE
#5  Idempotency                                  IN PROGRESS
#6  Payment provider boundary
#7  Moniepoint integration
#8  Payment failure / unknown / reconciliation
#9  Server-controlled fare
#10 Database transaction / integrity strategy
#11 NFC passenger card / credential
#12 Card lifecycle
#13 Driver operating session
#14 Fare rules
#15 Route model
#16 Driver receiving account
#17 Daily ticket
#18 Settlement
#19 Payment / trip history
#20 Versioned fee configuration
#21 Audit logs
#22 API error system
#23 Rate limiting
#24 Observability / logging
#25 Tests + security hardening + deployment readiness

THEN

#1 Authentication
#2 Authorization
```

This ordering is deliberate. Authentication and authorization will eventually protect these capabilities, but we first need the underlying payment, transport, state, audit and integrity model to be correct.

---

# 4. Fixing Track #3 to #25

## #3 Money Representation ~ COMPLETE

All monetary values use integer kobo.

```text
₦510.00 = 51000 kobo
₦500.00 = 50000 kobo
```

Database representation:

```text
BIGINT
```

Never use floating-point values for financial amounts.

Trip financial fields:

```text
fare_amount
passenger_platform_fee
driver_platform_fee
passenger_total
driver_payout
platform_total
```

Financial example:

```text
Fare                    ₦500
Passenger platform fee   ₦10
Passenger pays           ₦510
Driver platform fee      ₦10
Driver receives          ₦490
Platform revenue         ₦20
```

## #4 Payment / Transaction State Model ~ COMPLETE

Payment state:

```text
PENDING
PROCESSING
SUCCEEDED
FAILED
UNKNOWN
```

Trip state:

```text
PENDING
COMPLETED
CANCELLED
```

Settlement state:

```text
PENDING
PROCESSING
SUCCEEDED
FAILED
UNKNOWN
```

Reconciliation is separate from payment outcome.

Core rule:

```text
Only payment = SUCCEEDED
        ↓
can make a fare financially successful
```

`UNKNOWN` is never treated as `FAILED`.

## #5 Idempotency — IN PROGRESS

Payment entity:

```text
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

Database foundation is complete.

Rules:

- One idempotency key represents one payment attempt.
- Retrying the exact same attempt reuses the same key.
- A fallback attempt against another passenger account gets a new key.
- `trip_id` is not unique because one trip can have multiple payment attempts.
- The database unique constraint is the final duplicate-protection layer.

Next implementation:

```text
Request
  ↓
FindByIdempotencyKey
  ↓
Existing?
 ├── YES → return existing payment state
 └── NO  → create payment attempt
```

Concurrent requests must still be protected by the database unique constraint.

### Cross-Cutting Requirement: Concurrency Safety

Race-condition protection applies from Stage #5 onward and is not treated as a standalone feature.

Any operation that can be triggered concurrently must be designed and tested for concurrent execution, especially operations involving money, identity, credentials, state transitions, limits, and settlement.

Key areas include:

- Payment idempotency and duplicate requests
- Concurrent payment attempts
- Account-linking limits
- Successful-payment-per-trip invariant
- Daily-ticket charging
- Card state transitions
- Driver operating sessions
- Settlement and payout operations
- Concurrent trip creation
- Configuration/version consistency

Stage #10 is the dedicated system-wide concurrency and database integrity pass.

This stage covers:

- PostgreSQL transactions
- Atomic state transitions
- Unique constraints and indexes
- Row-level locking where required
- `SELECT ... FOR UPDATE` where appropriate
- Race-condition analysis
- Concurrent request testing
- Database/service invariant enforcement

The principle is:

> If two requests can arrive at the same time, the system must still produce one correct and deterministic outcome.

## #6 Payment Provider Boundary

Create an internal provider interface.

The business layer must not know Moniepoint HTTP details.

Conceptually:

```text
Payment Service
      ↓
PaymentProvider
      ↓
Moniepoint Adapter
      ↓
Moniepoint API
```

Before implementation, confirm Moniepoint's actual:

- authentication
- payment operation
- account/payment-source representation
- provider references
- success/failure/pending outcomes
- idempotency mechanism
- webhook/callback behavior
- timeout behavior
- reconciliation/status lookup

Do not invent the provider contract.

## #7 Moniepoint Integration

Implement the real Moniepoint adapter after the provider boundary is verified.

Required behavior:

- sandbox credentials separated from production
- provider references stored
- provider outcomes mapped to internal states
- provider errors translated into stable internal errors
- timeouts treated as unknown when appropriate
- retries use the correct idempotency key
- secrets never enter source control or logs

## #8 Payment Failure, Unknown State and Reconciliation

Required paths:

```text
SUCCESS
FAILURE
TIMEOUT / UNKNOWN
DATABASE FAILURE
PROVIDER SUCCESS + LOCAL FAILURE
```

Never automatically fall back to another account after an unknown result.

Safe fallback should be limited to explicitly safe outcomes, such as confirmed insufficient funds.

Unknown payment:

```text
Payment = UNKNOWN
Trip = PENDING
```

Then reconcile using the provider reference/status mechanism.

## #9 Server-Controlled Fare

The client must eventually stop supplying the authoritative fare.

Current prototype:

```text
client → amount_kobo
```

MVP target:

```text
terminal
   ↓
vehicle / route context
   ↓
fare service
   ↓
FareQuote
   ↓
payment authorization
```

The driver does not manually type fares.

The passenger sees the fare before confirming/tapping.

## #10 Database Transaction and Integrity Strategy

The financial tap flow must use real database transactions.

Target:

```text
BEGIN
  ↓
validate state
  ↓
create/update payment record
  ↓
record trip state
  ↓
commit
```

Failures roll back local database changes.

The design must also prevent:

- duplicate successful payment for one boarding event
- inconsistent payment/trip state
- duplicate daily ticket
- duplicate settlement
- account selection races
- invalid references

External payment calls cannot be magically rolled back by PostgreSQL, so provider reconciliation remains necessary.

## #11 NFC Passenger Card / Credential

The physical NFC card is a credential.

It is:

- not a bank account
- not a wallet
- not the source of funds
- not a payout destination

Relationship:

```text
NFC card
   ↓
Passenger identity
   ↓
Linked bank accounts
```

The card should identify the passenger without exposing financial credentials.

## #12 Card Lifecycle

States:

```text
ACTIVE
INACTIVE
BLOCKED
VOID
```

Allowed lifecycle:

```text
ACTIVE ↔ INACTIVE

ACTIVE / INACTIVE
       ↓
    BLOCKED
       ↓
      VOID
       ↓
replacement card → ACTIVE
```

Lost/stolen card:

```text
Passenger blocks old card
        ↓
Old card can no longer authorize taps
        ↓
Replacement issued
        ↓
New card bound to same passenger identity
```

Card state changes must be auditable.

## #13 Driver Operating Session

Terminal registration is not a driver session.

MVP requires:

```text
terminal binding
      +
active driver session
      +
session verification
```

Session should contain:

- driver identity
- terminal
- start time
- expiry
- verification method
- status

An expired session must not authorize new trips.

## #14 Fare Rules

Fare must become a server-side rule.

Supported future sources:

```text
explicit_rule
route_tier
distance_band
```

For MVP, a simple single-corridor fare is acceptable.

The important rule is that the fare comes from trusted server configuration, not a driver-entered amount.

## #15 Route Model

MVP can start with one pilot corridor:

```text
Tanke ↔ Challenge
```

The data model must leave room for:

```text
route
route_fare_tier
terminal_location
```

Formal multi-corridor route management can follow in T2.

## #16 Driver Receiving Account

Driver payout destination is server-resolved.

Never accept these from `/tap`:

```text
driver_bank_account
payout_destination
union_bank_account
```

For driver-controlled transport:

```text
Terminal
   ↓
Bound driver
   ↓
Driver receiving account
```

Driver must have a verified linked receiving account before payout.

## #17 Daily Driver Ticket

The daily ticket is a separate financial event.

Trigger:

```text
first successful passenger fare
        ↓
driver operating day begins
        ↓
check daily ticket
        ↓
if unpaid → debit driver receiving account
        ↓
pay union account
```

One ticket per driver per operating date.

Database invariant:

```text
UNIQUE(driver_id, operating_date)
```

Ticket failure must not automatically turn a successful passenger fare into a failed fare.

Unknown ticket outcome must never cause an automatic duplicate debit.

## #18 Settlement

Separate:

```text
payment
trip
settlement
```

Passenger fare authorization is not the same thing as driver settlement.

Target:

```text
Passenger payment
       ↓
successful collection
       ↓
settlement record
       ↓
driver payout
```

Settlement needs its own state and provider reference.

## #19 Payment / Trip History

Passenger history should show:

- trip
- fare
- passenger platform fee
- passenger total
- payment status
- payment provider reference where appropriate
- date/time

Driver history should show:

- fare earned
- driver platform fee
- driver payout
- daily ticket
- ticket status
- date/time

Never expose full bank account numbers.

## #20 Versioned Fee Configuration

Do not permanently hardcode:

```text
passenger fee = ₦10
driver fee    = ₦10
daily ticket  = ₦X
```

Configuration needs:

- effective dates
- versioning
- privileged changes
- audit trail
- historical accuracy

If the fee changes:

```text
Old trip → keeps old fee
New trip → uses new fee
```

Historical transactions must never be silently recalculated.

## #21 Audit Logs

Audit privileged and security-sensitive actions.

At minimum:

```text
card issued
card blocked
card voided
card reissued
terminal bound
terminal rebound
driver session actions
fee changes
daily ticket events
reconciliation actions
admin actions
```

Audit record should capture:

```text
who
what
when
entity
old state/value
new state/value
reason
```

## #22 API Error System

Stop returning ad-hoc error strings everywhere.

Introduce stable error categories/codes.

Examples:

```text
INVALID_REQUEST
NOT_FOUND
UNAUTHORIZED
FORBIDDEN
CARD_INACTIVE
CARD_BLOCKED
SESSION_EXPIRED
PAYMENT_FAILED
PAYMENT_UNKNOWN
DUPLICATE_REQUEST
INSUFFICIENT_FUNDS
RATE_LIMITED
INTERNAL_ERROR
```

Clients can react to codes reliably without parsing human text.

## #23 Rate Limiting

MVP rate limiting:

```text
/tap
/location
public endpoints
```

Key `/tap` limits by terminal/device identity where possible.

Public endpoints can additionally be limited by IP.

The goal is to prevent a compromised terminal or client from hammering the API.

## #24 Observability and Logging

Use structured logs.

Preferred Go approach:

```text
log/slog
```

Safe identifiers:

```text
trip_id
payment_id
terminal_id
session_id
credential_id
```

Never log:

- payment secrets
- session tokens
- full bank account numbers
- private signing keys
- raw biometric data

MVP operational basics:

- `/health`
- database health check
- error tracking
- useful structured logs
- uptime monitoring

## #25 Tests, Security Hardening and Deployment Readiness

### Unit tests

Test:

- fare calculations
- account priority
- card transitions
- session expiration
- idempotency
- duplicate taps
- payout resolution
- daily ticket uniqueness
- error mapping

### Integration tests

Test:

```text
tap
 ↓
credential
 ↓
session
 ↓
account selection
 ↓
payment
 ↓
trip
 ↓
settlement
```

Use mocked external payment/identity providers.

### Failure tests

Explicitly test:

- insufficient funds
- provider rejection
- provider timeout
- unknown payment
- duplicate idempotency key
- duplicate tap
- inactive card
- blocked card
- void card
- expired session
- inactive terminal
- database failure
- payment/database inconsistency
- duplicate daily ticket
- invalid card transition

### Security/deployment

Before pilot:

- no secrets in git
- separate sandbox/production credentials
- least-privilege Postgres user
- parameterized SQL
- TLS
- dependency vulnerability scanning
- nightly database backups
- `/health`
- uptime monitoring
- graceful shutdown
- CI build/vet/test
- Docker deployment
- structured/redacted logs

---

# 5. Authentication — After #25

Authentication is intentionally delayed until the system underneath it is coherent.

MVP authentication eventually needs:

```text
Passenger login
Driver login
Staff/agent login
Admin login
```

The authenticated identity must come from the server-side session/token.

Do not trust:

```text
passenger_id
driver_id
admin_id
```

from the client for authorization decisions.

---

# 6. Authorization — After Authentication

Authorization determines what the authenticated identity may do.

Examples:

```text
Passenger
  → own cards
  → own accounts
  → own trips

Driver
  → own operating session
  → own terminal
  → own trips
  → own receiving account

Agent
  → passenger enrollment
  → card issuance/reissue within assigned scope

Ops
  → operational management

Super Admin
  → privileged configuration and oversight
```

Admin authorization must distinguish at least:

```text
ops
super_admin
```

Privileged actions are audited.

---

# 7. MVP Pilot Surface

### Passenger

- enrollment
- linked bank accounts
- physical NFC card
- view fare
- tap/pay
- transaction history
- block lost card
- replacement workflow

### Driver

- login
- start operating session
- terminal status
- fare display
- passenger tap processing
- transaction history
- receiving account
- daily ticket visibility

### Field Agent

- authenticated agent mode
- passenger enrollment
- card issuance
- lost-card reissue
- customer support

### HQ / Ops

MVP monitoring surface:

- driver/session status
- terminal status
- trip/payment visibility
- provisioning/rebinding operations
- reconciliation visibility

A full investor dashboard is not required for MVP.

---

# 8. MVP Hardware

Start with budget Android devices that support:

- NFC
- fingerprint/local authentication where required
- GPS
- mobile connectivity

These run the Flutter driver terminal app.

PAX A920 Pro is a later fleet transition, not an MVP prerequisite.

Custom hardware is later.

---

# 9. MVP Success Criteria

The MVP is complete when this works reliably:

```text
1. Passenger is enrolled
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
```

---

# 10. What We Do NOT Build Yet

To protect MVP scope, defer:

```text
React investor dashboard
Deck.gl investor mode
USSD
BRT integration
Taxi partner API
Multi-state expansion
Multi-city scaling
Redis
Horizontal API scaling
Advanced PostGIS routing
OSRM route planning
Custom hardware
Full public partner API
```

These are valuable, but they do not need to block the first real cash-free pilot.

---

# 11. Migration Discipline

Continue following the engineering migration rules:

```text
0001_init.sql
0002_money_representation.sql
0003_payment_idempotency.sql
0004_next_logical_schema_change.sql
...
```

Rules:

- never renumber an existing migration
- one migration = one logical schema change
- additive by default
- destructive changes require explicit review and data-handling notes
- do not modify an already-applied migration to change history

Current migration state:

```text
0001  applied
0002  applied
0003  applied
```

---

# 12. Immediate Next Engineering Task

We are currently here:

```text
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

After #5 is solid:

```text
#6 PaymentProvider
     ↓
#7 Moniepoint
     ↓
#8 failure/reconciliation
```

Do not jump to Flutter or React until the backend payment path is structurally ready.

---

## Final MVP Principle

> **The MVP is not the smallest number of features. It is the smallest system that can move real money safely, identify the right passenger and driver, record what happened, recover from uncertainty, and explain every transaction afterwards.**
