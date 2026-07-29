-- 0001_init.sql
-- Matches the existing Go structs exactly: User, PassengerAccount, Terminal, Trip.
-- This is step 2 only — session/credential/card_batch/PostGIS come in later migrations.

CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY,
    phone       VARCHAR(15) UNIQUE NOT NULL,
    full_name   VARCHAR(100) NOT NULL,
    role        VARCHAR(20) NOT NULL, -- 'passenger' | 'driver' (extend later: 'agent', 'operator')
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS passenger_accounts (
    id              UUID PRIMARY KEY,
    passenger_id    UUID NOT NULL REFERENCES users(id),
    account_number  VARCHAR(20) NOT NULL,
    bank_code       VARCHAR(10) NOT NULL,
    bank_name       VARCHAR(60) NOT NULL,
    priority_order  INT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_passenger_accounts_passenger_id ON passenger_accounts(passenger_id);

CREATE TABLE IF NOT EXISTS terminals (
    id              UUID PRIMARY KEY,
    driver_id       UUID NOT NULL REFERENCES users(id),
    device_serial   VARCHAR(64) UNIQUE NOT NULL,
    vehicle_plate   VARCHAR(20) NOT NULL,
    terminal_type   VARCHAR(20) NOT NULL, -- 'keke' | 'bus'
    last_lat        DOUBLE PRECISION,
    last_lng        DOUBLE PRECISION,
    last_seen_at    TIMESTAMPTZ,
    is_active       BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS trips (
    id              UUID PRIMARY KEY,
    passenger_id    UUID NOT NULL REFERENCES users(id),
    driver_id       UUID NOT NULL REFERENCES users(id),
    terminal_id     UUID NOT NULL REFERENCES terminals(id),
    fare_amount     NUMERIC(10,2) NOT NULL,
    driver_payout   NUMERIC(10,2) NOT NULL,
    platform_fee    NUMERIC(10,2) NOT NULL,
    pickup_lat      DOUBLE PRECISION,
    pickup_lng      DOUBLE PRECISION,
    status          VARCHAR(20) NOT NULL,
    tapped_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trips_passenger_id ON trips(passenger_id);
CREATE INDEX IF NOT EXISTS idx_trips_driver_id ON trips(driver_id);

-- Least-privilege API role (security checklist item T0)
-- Run these manually as a superuser once, then use this role in your DSN:
-- CREATE ROLE transport_api WITH LOGIN PASSWORD 'change-me';
-- GRANT SELECT, INSERT, UPDATE ON users, passenger_accounts, terminals, trips TO transport_api;
