# Alabastr addittional  — Engineering Guide

> Internal engineering guide. Deeper implementation guidance belongs here rather than in the public README.

## 1. Backend Authority

Go + Gin owns authentication, authorization, identity resolution, credential validation, terminal validation, fare authorization, payment orchestration, payout resolution, trip persistence and audit events.

Flutter and React are clients. They must not become alternate business-logic engines.

## 2. Repository Boundary

Database access stays behind interfaces:

```text
Handler
   ↓
Service / use case
   ↓
Repository interface
   ↓
PostgreSQL implementation
```

Current repositories cover users, passenger accounts, terminals and trips. SQL remains parameterized.

## 3. Identity and Linked Accounts

A passenger identity is separate from financial accounts.

A passenger may have up to three linked bank accounts. `priority_order` controls fallback order.

These accounts are **not** a proprietary wallet.

## 4. Transport Credentials

The company-issued physical NFC card is a credential belonging to a passenger identity.

Do not represent its entire lifecycle with one ambiguous boolean.

Recommended states:

```text
ACTIVE
INACTIVE
BLOCKED
VOID
```

Meaning:

- `ACTIVE`: may authorize a tap.
- `INACTIVE`: temporarily disabled by passenger.
- `BLOCKED`: security action for lost/stolen/compromised card; cannot be reactivated normally.
- `VOID`: permanently retired, normally after replacement.

## 5. Card State Machine

```text
                    ACTIVE
                       │
          passenger turns card off
                       ↓
                    INACTIVE
                       │
           passenger turns card on
                       ↓
                    ACTIVE

                    ACTIVE
                       │
                lost / stolen
                       ↓
                    BLOCKED
                       │
                  replacement
                       ↓
                OLD CARD VOID
                       │
                       ↓
                 NEW CARD ACTIVE
```

Card state is independent from bank-account state and passenger identity state.

## 6. Card Activation API

The passenger controls the state of the **company-issued transport card**, not the linked bank accounts. The card is simply the credential that authorizes transport payment against those accounts.

Conceptually:

```text
GET  /cards/:id
POST /cards/:id/activate
POST /cards/:id/deactivate
POST /cards/:id/block
POST /cards/:id/reissue
```

Exact route names may change.

Passenger actions must use the authenticated identity from middleware rather than a client-supplied passenger ID:

```go
router.POST("/cards/:id/deactivate", RequireRole("passenger"), app.deactivateCardHandler)
router.POST("/cards/:id/block", RequireRole("passenger"), app.blockCardHandler)
```

The backend must verify that the credential belongs to the authenticated passenger.

### Activation semantics

- `activate` is valid only for the passenger's current `INACTIVE` credential.
- `deactivate` temporarily disables the current card and is reversible by the passenger.
- `block` is a security action for a lost, stolen or compromised card and takes effect immediately.
- A `BLOCKED` or `VOID` card cannot be reactivated through the normal passenger activation endpoint.
- Card state changes do **not** modify, freeze or activate any linked bank account.
- State-changing operations should be idempotent where practical so repeated requests do not create conflicting events.

Every state transition should produce an auditable card event containing the credential, passenger identity, previous state, new state, actor, reason and timestamp.

## 7. Card Authorization Rules

Before payment:

```text
Find credential
    ↓
Does it exist?
    ↓
Resolve identity
    ↓
Is state ACTIVE?
    ↓
Validate terminal
    ↓
Validate driver session
    ↓
Validate fare
    ↓
Resolve payment source
```

If state is `INACTIVE`, `BLOCKED` or `VOID`, reject before payment is attempted.

## 8. Card Reissue

Reissue is security-sensitive because it creates a new physical credential for an existing passenger identity.

### Passenger self-service

A lost/stolen report should immediately move the current card to `BLOCKED`. This is deliberately separate from reissue: **blocking protects the account immediately; replacement happens afterward.**

```text
Passenger app
     ↓
Authenticate passenger
     ↓
Verify current credential belongs to passenger
     ↓
BLOCK old card immediately
     ↓
Replacement request / status
```

The passenger should be able to see that replacement is pending without exposing the raw card UID.

### Physical replacement

```text
Passenger
    ↓
identity verification
    ↓
old credential located
    ↓
confirm old credential is BLOCKED / eligible for replacement
    ↓
new card UID assigned
    ↓
old credential → VOID
    ↓
new credential created for same identity
    ↓
new credential → ACTIVE
    ↓
issuance + replacement event recorded
```

The replacement operation should be atomic from the application's perspective: it must not leave two active physical cards for the same passenger. If issuance fails, preserve the old blocked state and do not silently activate a partial replacement.

Never delete the old credential. Preserve it for audit/history while making it permanently unable to authorize.

### Important invariant

Under the normal physical-card model, only the current card should be capable of authorizing payment for the passenger. A passenger may have historical `VOID` credentials, but there should normally be at most one `ACTIVE` physical card credential at a time.

## 9. Card Issuance Audit

Eventually record:

```text
credential_id
identity_id
issued_by
issued_at
reason
previous_credential_id
previous_status
new_status
blocked_reason
replacement_request_id
```

For agent operations also record:

```text
agent_id
location
batch_id
```

## 10. Tap Endpoint

The current `/tap` implementation is a prototype. The production use case should orchestrate:

```text
Tap request
   │
   ├── Parse and validate request
   ├── Authenticate terminal/device
   ├── Resolve credential
   ├── Validate credential state
   ├── Validate passenger identity
   ├── Validate terminal
   ├── Validate terminal active state
   ├── Validate driver session
   ├── Resolve fare
   ├── Resolve passenger linked accounts
   ├── Resolve driver/company receiving account
   ├── Attempt payment in priority order
   ├── Apply platform fee
   ├── Persist trip
   └── Return transaction result
```

## 11. Passenger Account Fallback

Do not treat `accounts[0]` as the complete production payment mechanism.

Conceptually:

```go
for _, account := range accounts {
    result := payment.Authorize(account, amount)
    if result.Success {
        return result
    }
    if result.IsInsufficientFunds() {
        continue
    }
    return result.Error
}
```

Only explicitly safe fallback conditions should proceed to the next account. A timeout or unknown payment state must not automatically fall through.

## 12. Driver / Company Receiving Account

Never accept the payout destination from the tap request.

Driver-controlled transport:

```text
terminal
   ↓
bound driver
   ↓
driver receiving account
```

Company-controlled transport:

```text
terminal
   ↓
operator/company
   ↓
company receiving account
```

This distinction supports future BRT/operator-controlled modes without corrupting the Keke/bus driver model.

## 13. Payment Processor Interface

External payment providers must sit behind an interface:

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

This enables sandboxing, provider replacement, unit testing and failure simulation.

## 14. Idempotency

Nonce replay and payment idempotency solve different problems.

**Nonce replay** prevents the exact signed request from being processed twice.

**Payment idempotency** prevents a retried provider call from creating two transfers.

Both are required.

The tap nonce should eventually be the stable idempotency key for that authorization attempt.

## 15. Duplicate Tap Protection

A second tap can be genuinely different while still representing the same boarding event.

Therefore use:

```text
nonce replay check
        +
short-window passenger + terminal duplicate check
```

A recent successful matching trip should be returned rather than charging again.

## 16. Transaction Boundaries

Do not assume a PostgreSQL rollback can undo an external bank transfer.

External payment operations are not PostgreSQL transactions.

Production handling must explicitly cover:

- provider timeout
- unknown provider result
- webhook confirmation
- idempotent retry
- reconciliation
- successful payment followed by database failure
- successful database write after provider uncertainty

## 17. Fare Resolution

Fare is determined before authorization:

```text
route
  ↓
destination
  ↓
GetFare
  ↓
FareQuote
  ↓
AuthorizeTrip(quote)
```

Supported sources:

```text
explicit_rule
route_tier
distance_band
```

The trip must retain the fare source so historical transactions remain explainable after future fare changes.

## 18. Terminal Architecture

A terminal is a physical NFC reader.

It may collect NFC credential data, GPS, connectivity state, device identity and signed request data.

It must not contain editable payout configuration, arbitrary destination accounts, authoritative fee logic or a secret that permits arbitrary fund routing.

## 19. Terminal and Vehicle Types

Current values:

```text
terminal_type = nfc_reader
vehicle_type  = keke | bus
```

Keep these fields separate.

Train is intentionally deferred because the operational model is company/operator controlled.

## 20. Driver Sessions

Terminal registration is not equivalent to an indefinitely valid driver session.

Eventually authorization requires:

```text
terminal binding
      +
active driver session
      +
session verification
```

Session should carry driver identity, terminal, route, start time, expiry, verification method and status.

## 21. Authentication and Middleware

Do not scatter authorization checks through handlers.

Use middleware:

```go
router.POST("/tap", RequireRole("driver"), app.tapHandler)
```

Admin authorization must distinguish at least:

```text
ops
super_admin
```

Privileged operations must be audited.

There is no public admin registration endpoint.

## 22. Secrets

Never hardcode database passwords, payment keys, identity-provider credentials, webhook secrets, signing secrets or production credentials.

Development may use an environment variable:

```powershell
$env:DATABASE_URL="postgres://USER:PASSWORD@localhost:5433/transportation_model?sslmode=disable"
```

The connection string is supplied to the process rather than committed to source control. Production should use proper secret management.

## 23. Logging

Never log:

- session tokens
- payment secrets
- full account numbers
- raw biometric data
- private signing keys

Use safe identifiers such as session ID, transaction ID, trip ID, terminal ID and credential ID.

## 24. External Integrations

Use interfaces for external services such as:

```text
IdentityVerifier
PaymentProcessor
CardIssuer
NotificationService
RouteProvider
```

The purpose is testability and controlled failure handling, not abstraction for its own sake.

## 25. Testing Strategy

### Unit tests

Test card state transitions and audit events, account priority selection, fare logic, middleware, duplicate-tap rules, nonce handling and payout resolution.

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
```

Use mocked external payment and identity providers.

### Failure tests

Explicitly test insufficient funds, provider timeout, provider rejection, duplicate nonce, duplicate tap, inactive card, blocked card, void card, unknown card, duplicate active-card creation, invalid card transition, inactive terminal, expired session, database failure and payment/database inconsistency.

## 26. Offline Resilience

Offline support must be designed rather than bolted on.

A client may eventually queue signed requests, but the backend must still enforce replay, stale-request, duplicate-submission and terminal-compromise rules.

## 27. Card Operations and Accountability

Eventually implement:

```text
Card inventory
      ↓
Batch allocation
      ↓
Agent issuance
      ↓
Passenger credential
      ↓
Activation
      ↓
Deactivation
      ↓
Block
      ↓
Reissue
      ↓
Reconciliation
```

Agents must not be able to silently create unlimited cards. Card stock must be accountable.

## 28. Flutter Direction

Flutter has not yet been implemented in the current codebase.

When it begins, passenger functionality should include:

```text
passenger_home/
    linked_accounts/
    card/
        status
        activate
        deactivate
        block
        replacement_status
    trip_history/
```

Driver functionality remains separate. HCE is later scope.

## 29. React Direction

React is for the administrative web surface: live map, provisioning, driver oversight, passenger/credential oversight, card issuance, card reconciliation, route management, fare management, financial reconciliation and audit review.

Privileged mutations remain authorized by Go.

## 30. HCE Future Path

A future passenger app may emulate the transport credential on Android.

Preferred security model:

```text
reader challenge
      ↓
phone private key
      ↓
signed response
      ↓
backend verification
```

Do not make a static identifier the security mechanism. Physical cards remain the universal fallback.

## 31. Engineering Sequence

Unless a dependency requires otherwise:

```text
1. PostgreSQL foundation
2. Repository boundaries
3. Terminal + passenger account persistence
4. Tap validation
5. Payment interfaces
6. Driver receiving account
7. Tap authorization
8. Idempotency + replay protection
9. Driver sessions
10. Authentication middleware
11. Passenger card schema + card event/audit history
12. Card activation/deactivation
13. Card blocking
14. Card reissue and replacement workflow
15. Card inventory/reconciliation
16. Flutter passenger/driver interfaces
17. React operations dashboard
18. Geospatial layer
19. Advanced fare engine
20. Partner integrations
21. Multi-city infrastructure
```

Do not implement a later layer merely because its architecture has already been designed.

## 32. Definition of Done

A feature is not done because the happy path works.

Ask:

```text
Can it authenticate the caller?
Can it reject unauthorized access?
Can it be retried safely?
Can it fail halfway?
Can it recover?
Can it be audited?
Can it be tested without external services?
Can the database remain consistent?
Can the user understand the result?
Can an operator investigate it later?
```

For payment features additionally ask:

```text
Can it charge twice?
Can money move to the wrong destination?
What happens when the provider times out?
What happens when the response is lost?
What happens when the database fails?
What happens when the card is blocked immediately before a tap?
Can a replacement ever leave two active cards?
Is the old credential retained but permanently unable to authorize?
```
