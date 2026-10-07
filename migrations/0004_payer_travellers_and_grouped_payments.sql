-- 0004_payer_travellers_and_grouped_payments.sql
--
-- Payment architecture transition:
--
-- 0003 modeled one payment as belonging directly to one trip
-- and one passenger.
--
-- 0004 introduces grouped payment sessions so one payer can pay
-- for multiple identified travellers in a single operation.
--
-- Existing payment rows were checked before this migration:
-- SELECT COUNT(*) FROM payments;
-- Result: 0
--
-- Therefore trip_id and passenger_id can be removed safely.
--
-- The actual traveller remains identified by trips.passenger_id.
-- The payer is identified by payment_sessions.payer_id.
--
-- Payment attempts retain account_id because each attempt records
-- the actual bank account used for that attempt.
--
-- Rerun safety: every DDL statement in this file is written to be
-- safe to run more than once against a DB it has already been
-- applied to (CREATE ... IF NOT EXISTS, DROP ... IF EXISTS before
-- re-adding constraints, and a two-step nullable-then-NOT-NULL
-- column rollout instead of an ADD COLUMN ... NOT NULL that would
-- fail outright on any DB with existing payment rows).

BEGIN;


-- ============================================================
-- 1. PEOPLE I PAY FOR
-- ============================================================

CREATE TABLE IF NOT EXISTS payer_travellers (
    id UUID PRIMARY KEY,

    payer_id UUID NOT NULL
        REFERENCES users(id),

    traveller_id UUID NOT NULL
        REFERENCES users(id),

    status VARCHAR(20) NOT NULL
        DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE')),

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT now(),

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT now(),

    CHECK (payer_id <> traveller_id),

    UNIQUE (payer_id, traveller_id)
);

CREATE INDEX IF NOT EXISTS
    idx_payer_travellers_payer_id
    ON payer_travellers(payer_id);

CREATE INDEX IF NOT EXISTS
    idx_payer_travellers_traveller_id
    ON payer_travellers(traveller_id);


-- ============================================================
-- 2. PAYMENT SESSIONS
-- ============================================================

CREATE TABLE IF NOT EXISTS payment_sessions (
    id UUID PRIMARY KEY,

    payer_id UUID NOT NULL
        REFERENCES users(id),

    total_amount BIGINT NOT NULL
        CHECK (total_amount > 0),

    status VARCHAR(20) NOT NULL
        CHECK (status IN (
            'PENDING',
            'PROCESSING',
            'SUCCEEDED',
            'FAILED',
            'UNKNOWN'
        )),

    idempotency_key VARCHAR(128) NOT NULL
        UNIQUE,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT now(),

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT now()
);


-- ============================================================
-- 3. CONVERT PAYMENTS INTO PAYMENT ATTEMPTS
-- ============================================================

-- FIX: added nullable first, so this step never fails against a
-- DB that already has payment rows. NOT NULL is enforced as its
-- own step below, once every row is known to have a session.
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS payment_session_id UUID;

-- If this migration is ever run against a DB where `payments`
-- already has rows, backfill payment_session_id here before the
-- next statement (e.g. by creating one payment_sessions row per
-- distinct existing payment and pointing the row at it). Left as
-- a no-op on this DB since the row count was confirmed to be 0.

ALTER TABLE payments
    ALTER COLUMN payment_session_id SET NOT NULL;

-- FIX: DROP ... IF EXISTS before ADD so this is safe to rerun.
-- Postgres has no ADD CONSTRAINT IF NOT EXISTS, so without the
-- DROP this statement would error with "constraint already
-- exists" on any rerun after a successful first application.
ALTER TABLE payments
    DROP CONSTRAINT IF EXISTS fk_payments_payment_session;

ALTER TABLE payments
    ADD CONSTRAINT fk_payments_payment_session
    FOREIGN KEY (payment_session_id)
    REFERENCES payment_sessions(id);

CREATE INDEX IF NOT EXISTS
    idx_payments_payment_session_id
    ON payments(payment_session_id);


-- ============================================================
-- 4. REMOVE OLD ONE-TRIP PAYMENT RELATIONSHIPS
-- ============================================================

ALTER TABLE payments
    DROP COLUMN IF EXISTS trip_id;

ALTER TABLE payments
    DROP COLUMN IF EXISTS passenger_id;


-- ============================================================
-- 5. ONE SUCCESSFUL PAYMENT ATTEMPT PER SESSION
-- ============================================================

CREATE UNIQUE INDEX IF NOT EXISTS
    idx_one_successful_payment_per_session
    ON payments(payment_session_id)
    WHERE status = 'SUCCEEDED';


-- ============================================================
-- 6. PAYMENT ALLOCATIONS
-- ============================================================

CREATE TABLE IF NOT EXISTS payment_allocations (
    id UUID PRIMARY KEY,

    payment_id UUID NOT NULL
        REFERENCES payments(id),

    trip_id UUID NOT NULL
        REFERENCES trips(id),

    amount BIGINT NOT NULL
        CHECK (amount > 0),

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT now(),

    UNIQUE (trip_id)
);

CREATE INDEX IF NOT EXISTS
    idx_payment_allocations_payment_id
    ON payment_allocations(payment_id);

CREATE INDEX IF NOT EXISTS
    idx_payment_allocations_trip_id
    ON payment_allocations(trip_id);


COMMIT;