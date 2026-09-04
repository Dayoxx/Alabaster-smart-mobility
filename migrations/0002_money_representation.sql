-- 0002_money_representation.sql
-- Convert trip monetary values from NGN NUMERIC to kobo BIGINT.
-- Also separate passenger and driver platform fees.

ALTER TABLE trips
    ADD COLUMN passenger_platform_fee_kobo BIGINT,
    ADD COLUMN driver_platform_fee_kobo BIGINT,
    ADD COLUMN passenger_total_kobo BIGINT,
    ADD COLUMN platform_total_kobo BIGINT;

-- The original prototype used:
-- passenger platform fee = ₦10
-- driver platform fee = ₦10
--
-- Existing fare_amount already represents the fare after removing
-- the passenger fee, so we preserve it and convert it to kobo.

UPDATE trips
SET
    passenger_platform_fee_kobo = 1000,
    driver_platform_fee_kobo = 1000,
    passenger_total_kobo = ROUND((fare_amount + 10.00) * 100),
    platform_total_kobo = ROUND(platform_fee * 100);

-- Convert the existing driver payout to kobo.
ALTER TABLE trips
    ADD COLUMN fare_amount_kobo BIGINT,
    ADD COLUMN driver_payout_kobo BIGINT;

UPDATE trips
SET
    fare_amount_kobo = ROUND(fare_amount * 100),
    driver_payout_kobo = ROUND(driver_payout * 100);

-- Remove old monetary columns.
ALTER TABLE trips
    DROP COLUMN fare_amount,
    DROP COLUMN driver_payout,
    DROP COLUMN platform_fee;

-- Rename the new columns to the final names.
ALTER TABLE trips
    RENAME COLUMN fare_amount_kobo TO fare_amount;

ALTER TABLE trips
    RENAME COLUMN driver_payout_kobo TO driver_payout;

ALTER TABLE trips
    RENAME COLUMN passenger_platform_fee_kobo TO passenger_platform_fee;

ALTER TABLE trips
    RENAME COLUMN driver_platform_fee_kobo TO driver_platform_fee;

ALTER TABLE trips
    RENAME COLUMN passenger_total_kobo TO passenger_total;

ALTER TABLE trips
    RENAME COLUMN platform_total_kobo TO platform_total;

-- Money cannot be negative.
ALTER TABLE trips
    ADD CONSTRAINT trips_fare_amount_non_negative
        CHECK (fare_amount >= 0),
    ADD CONSTRAINT trips_passenger_platform_fee_non_negative
        CHECK (passenger_platform_fee >= 0),
    ADD CONSTRAINT trips_driver_platform_fee_non_negative
        CHECK (driver_platform_fee >= 0),
    ADD CONSTRAINT trips_passenger_total_non_negative
        CHECK (passenger_total >= 0),
    ADD CONSTRAINT trips_driver_payout_non_negative
        CHECK (driver_payout >= 0),
    ADD CONSTRAINT trips_platform_total_non_negative
        CHECK (platform_total >= 0);