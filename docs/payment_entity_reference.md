# Payment Entity — Design Reference

Reference doc for the `payments` table / `Payment` model design, captured ahead of writing `0003_payment_idempotency.sql`.

## Context: `trips` table state check

Before touching payments, we confirmed the migration status of the `trips` table money columns via `\d trips`:

```
                           Table "public.trips"
    Column     |           Type           | Collation | Nullable | Default
---------------+--------------------------+-----------+----------+---------
 id            | uuid                     |           | not null |
 passenger_id  | uuid                     |           | not null |
 driver_id     | uuid                     |           | not null |
 terminal_id   | uuid                     |           | not null |
 fare_amount   | numeric(10,2)            |           | not null |
 driver_payout | numeric(10,2)            |           | not null |
 platform_fee  | numeric(10,2)            |           | not null |
 pickup_lat    | double precision         |           |
 pickup_lng    | double precision         |           |
 status        | character varying(20)    |           | not null |
 tapped_at     | timestamp with time zone |           | not null | now()
Indexes:
    "trips_pkey" PRIMARY KEY, btree (id)
    "idx_trips_driver_id" btree (driver_id)
    "idx_trips_passenger_id" btree (passenger_id)
Foreign-key constraints:
    "trips_driver_id_fkey" FOREIGN KEY (driver_id) REFERENCES users(id)
    "trips_passenger_id_fkey" FOREIGN KEY (passenger_id) REFERENCES users(id)
    "trips_terminal_id_fkey" FOREIGN KEY (terminal_id) REFERENCES terminals(id)
```

**Result:** still on the old `numeric(10,2)` money columns (`fare_amount`, `driver_payout`, `platform_fee`) — the bigint-cents migration (breakdown into `passenger_platform_fee`, `driver_platform_fee`, `passenger_total`, `driver_payout`, `platform_total`) has **not** happened yet. Confirmed with a fresh `\d trips`, not just re-running the migration.

## Design discussion: is `passenger_id` on `payment` redundant?

Since `payment.trip_id → trips.passenger_id`, the passenger is technically derivable through the trip, so a strict normalization would drop `passenger_id` from `payment`.

**Normalized version (rejected):**
```
payment
├── id
├── trip_id
├── account_id
├── amount
├── status
├── idempotency_key
├── provider
├── provider_reference
├── created_at
└── updated_at
```

**Decision: keep `passenger_id` anyway.** Reasoning:
- Payment history is a major user-facing feature; keeping the passenger attached directly makes the financial record self-describing.
- Avoids a join for a common query shape:
  ```sql
  SELECT * FROM payments WHERE passenger_id = $1;
  ```
- Trade-off accepted: must maintain the invariant `payments.passenger_id = trips.passenger_id` through discipline (app-level, not enforced by a constraint here).

## Approved `payment` entity structure

```
payment
├── id                    ← payment identity
├── trip_id               ← which trip this payment belongs to
├── passenger_id          ← who paid (kept despite trip_id redundancy see above)
├── account_id            ← which linked account was attempted
├── amount                ← amount attempted/charged
├── status                ← payment lifecycle
├── idempotency_key       ← prevents duplicate processing
├── provider              ← Moniepoint
├── provider_reference    ← Moniepoint's transaction ID
├── created_at            ← when we created it
└── updated_at            ← last state change
```

Scope note: no driver_id, vehicle_plate, bank_name, fare, etc. on `payment` those belong to their own entities (trip, driver, account).

## Where this structure lives

The column list above describes **database columns**, not files or folders it does *not* map to a directory like `internal/payment/id/`, `internal/payment/trip_id/`, etc.

It's represented in two places:

```
DATABASE
migrations/0003_payment_idempotency.sql
        │  defines columns
        ▼
    payments table

GO APPLICATION
internal/models/payment.go
        │  represents the row
        ▼
    Payment struct
```

`Payment` struct (Go) will roughly mirror the table:

```
Payment
├── ID
├── TripID
├── PassengerID
├── AccountID
├── Amount
├── Status
├── IdempotencyKey
├── Provider
├── ProviderReference
├── CreatedAt
└── UpdatedAt
```

## Next step

Design `payments` in `migrations/0003_payment_idempotency.sql` first — including foreign keys, data types, constraints, and the unique idempotency constraint — before writing the Go model, so the database constraints drive the model rather than the other way around.
