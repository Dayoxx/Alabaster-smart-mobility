# Alabastr — Payment / Transaction State Model

## #4 — Payment / Transaction State Model

### Purpose

The current prototype uses `trips.status = "success"` in a way that can incorrectly imply three different things:

1. The passenger's payment succeeded.
2. The trip happened.
3. The money owed to the driver/platform was settled.

These are separate lifecycles and must be modeled independently.

## 1. Payment Status

Payment status answers: **Did the passenger's money actually get authorized/charged?**

| State | Meaning |
|---|---|
| `PENDING` | Payment record exists, but processing has not finished. |
| `PROCESSING` | Payment request has been sent to the payment provider. |
| `SUCCEEDED` | Provider confirms the passenger was charged successfully. |
| `FAILED` | Provider definitively says the charge failed. |
| `UNKNOWN` | The provider outcome is uncertain and requires reconciliation. |

**Critical rule:** `UNKNOWN` is not the same as `FAILED`.

If a provider times out after receiving a payment request, the system cannot safely assume the payment failed. The passenger's account may already have been charged. Therefore, an `UNKNOWN` payment must not automatically fall through to another linked bank account.

## 2. Trip Status

Trip status answers: **What happened to the passenger's journey?**

| State | Meaning |
|---|---|
| `PENDING` | Trip/payment process has started. |
| `COMPLETED` | Passenger's trip transaction successfully completed. |
| `CANCELLED` | Trip was cancelled. |

The model can later be expanded with journey-specific states such as `IN_PROGRESS` and `ENDED` when actual journey tracking is introduced.

For the current tap model, a confirmed successful payment allows the trip to become completed.

## 3. Settlement Status

Settlement status answers: **Has the money owed to the driver/platform actually been settled?**

| State | Meaning |
|---|---|
| `PENDING` | Settlement is waiting to happen. |
| `PROCESSING` | Settlement request is being processed. |
| `SUCCEEDED` | Recipient has been successfully credited/settled. |
| `FAILED` | Settlement definitively failed. |
| `UNKNOWN` | Settlement provider outcome is uncertain. |

Payment success and settlement success are separate events.

A valid state can therefore be:

```text
Payment:    SUCCEEDED
Trip:       COMPLETED
Settlement: PENDING
```

The passenger has paid and the trip has completed, but the driver's money has not yet been settled.

## 4. Core Payment Flow

```text
Passenger taps
      |
      v
Create payment
      |
      v
PENDING
      |
      v
PROCESSING
      |
      +---------------> FAILED
      |
      +---------------> UNKNOWN
      |
      v
  SUCCEEDED
      |
      v
Trip COMPLETED
      |
      v
Settlement PENDING
      |
      v
Settlement PROCESSING
      |
      +---------------> FAILED
      |
      +---------------> UNKNOWN
      |
      v
Settlement SUCCEEDED
```

## 5. Important Failure Scenarios

### Case 1 — Payment fails

```text
Payment:    FAILED
Trip:       CANCELLED
Settlement: NOT_DUE
```

There should be no successful driver payout and no platform revenue from the failed fare.

### Case 2 — Payment succeeds

```text
Payment:    SUCCEEDED
Trip:       COMPLETED
Settlement: PENDING
```

The passenger paid successfully and the trip completed. Settlement can happen separately.

### Case 3 — Provider times out

```text
Payment:    UNKNOWN
Trip:       PENDING
Settlement: NOT_DUE
```

Do **not** automatically try the next bank account.

The first payment attempt may have succeeded even though the response was lost or timed out.

This scenario will be handled through idempotency and reconciliation.

### Case 4 — Provider succeeds but database write fails

Example:

```text
Moniepoint -> SUCCESS
PostgreSQL  -> ERROR
```

The system must not simply classify the payment as failed. The passenger may already have been charged.

This is why every payment needs a durable internal record, provider reference, idempotency strategy, and eventual reconciliation mechanism.

## 6. Separation of Responsibilities

The system should move away from putting all financial meaning inside `trips`.

Conceptually:

```text
users
   |
   +-- passenger_accounts
   |
   +-- cards

trips
   |
   +-- payments
          |
          +-- settlements
```

### `trips`
Answers: **What transport event happened?**

### `payments`
Answers: **What happened to the passenger's money?**

### `settlements`
Answers: **What happened to the money owed to recipients?**

This separation is important because a trip can be completed while settlement is still pending.

## 7. Reconciliation Is Not a Payment Outcome

Reconciliation should not replace the payment status.

Instead, payment outcome and reconciliation should be separate concepts.

For example:

```text
payment_status = SUCCEEDED
reconciliation_status = RECONCILED
```

is clearer than:

```text
payment_status = RECONCILED
```

Payment status tells us what happened to the payment.

Reconciliation status tells us whether our records have been compared against the provider's records and confirmed.

### Future reconciliation states

```text
NOT_REQUIRED
PENDING
RECONCILED
DISPUTED
```

Detailed reconciliation behavior belongs to **#8 — Payment Failure / Unknown / Reconciliation**.

## 8. Core Invariant

The central financial rule is:

> **Only `payment = SUCCEEDED` can make a fare financially successful.**

The following must not independently be treated as proof of successful payment:

```text
HTTP 200
Trip created
Payment request sent
Provider request accepted
No immediate provider error
```

Only a confirmed successful payment outcome qualifies.

## 9. Combined State Model

```text
                 +--------------+
                 |     TRIP     |
                 |              |
                 | PENDING      |
                 | COMPLETED    |
                 | CANCELLED    |
                 +------+-------+
                        |
                 +------v-------+
                 |   PAYMENT    |
                 |              |
                 | PENDING      |
                 | PROCESSING   |
                 | SUCCEEDED    |
                 | FAILED       |
                 | UNKNOWN      |
                 +------+-------+
                        |
                 +------v-------+
                 |  SETTLEMENT  |
                 |              |
                 | PENDING      |
                 | PROCESSING   |
                 | SUCCEEDED    |
                 | FAILED       |
                 | UNKNOWN      |
                 +--------------+
```

## 10. Implementation Direction

No implementation code should be added until these state rules are agreed upon.

The next stages are:

- **#5 — Idempotency**
- **#6 — Payment Provider Boundary**
- **#7 — Moniepoint Integration**
- **#8 — Payment Failure / Unknown / Reconciliation**

These stages depend on the state model defined here.

## Current Architectural Decision

The platform will treat:

- **Trip** as the transport event.
- **Payment** as the passenger financial transaction.
- **Settlement** as the movement of money owed to recipients.
- **Reconciliation** as the process of confirming internal records against provider records.

These concepts must not be collapsed into one generic `status`.
