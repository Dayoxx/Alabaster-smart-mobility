-- 0005_trip_financial_invariants.sql
--
-- Enforce trip financial business invariants.
--
-- fare and platform fees must be positive.
-- Monetary totals must remain internally consistent.

BEGIN;

ALTER TABLE trips
    DROP CONSTRAINT IF EXISTS trips_fare_amount_non_negative,
    DROP CONSTRAINT IF EXISTS trips_passenger_platform_fee_non_negative,
    DROP CONSTRAINT IF EXISTS trips_driver_platform_fee_non_negative,
    DROP CONSTRAINT IF EXISTS trips_passenger_total_non_negative,
    DROP CONSTRAINT IF EXISTS trips_driver_payout_non_negative,
    DROP CONSTRAINT IF EXISTS trips_platform_total_non_negative;

ALTER TABLE trips
    ADD CONSTRAINT trips_fare_amount_positive
        CHECK (fare_amount > 0),

    ADD CONSTRAINT trips_passenger_platform_fee_positive
        CHECK (passenger_platform_fee > 0),

    ADD CONSTRAINT trips_driver_platform_fee_positive
        CHECK (driver_platform_fee > 0),

    ADD CONSTRAINT trips_passenger_total_positive
        CHECK (passenger_total > 0),

    ADD CONSTRAINT trips_driver_payout_positive
        CHECK (driver_payout > 0),

    ADD CONSTRAINT trips_platform_total_positive
        CHECK (platform_total > 0),

    ADD CONSTRAINT trips_passenger_total_matches_components
        CHECK (
            passenger_total =
            fare_amount + passenger_platform_fee
        ),

    ADD CONSTRAINT trips_driver_payout_matches_components
        CHECK (
            driver_payout =
            fare_amount - driver_platform_fee
        ),

    ADD CONSTRAINT trips_platform_total_matches_components
        CHECK (
            platform_total =
            passenger_platform_fee + driver_platform_fee
        );

COMMIT;