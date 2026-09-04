-- 0003_payment_idempotency.sql

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,

    trip_id UUID NOT NULL
        REFERENCES trips(id),

    passenger_id UUID NOT NULL
        REFERENCES users(id),

    account_id UUID NOT NULL
        REFERENCES passenger_accounts(id),

    amount BIGINT NOT NULL
        CHECK (amount > 0),

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

    provider VARCHAR(30) NOT NULL,

    provider_reference VARCHAR(128),

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT now(),

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS
    idx_payments_provider_reference
    ON payments(provider, provider_reference)
    WHERE provider_reference IS NOT NULL;

CREATE INDEX IF NOT EXISTS
    idx_payments_trip_id
    ON payments(trip_id);

CREATE INDEX IF NOT EXISTS
    idx_payments_passenger_id
    ON payments(passenger_id);

CREATE INDEX IF NOT EXISTS
    idx_payments_account_id
    ON payments(account_id);