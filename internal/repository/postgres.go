package repository

import (
	"context"
	"database/sql"
	"errors"
	"transportation/internal/models"

	"github.com/google/uuid"
)

type PostgresUserRepository struct{ db *sql.DB }

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// createuser
func (r *PostgresUserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, phone, full_name, role, created_at) VALUES ($1, $2, $3, $4, $5)`,
		user.ID, user.Phone, user.FullName, user.Role, user.CreatedAt)
	return user, err
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	var user models.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, phone, full_name, role, created_at FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Phone, &user.FullName, &user.Role, &user.CreatedAt)
	return user, err
}

type PostgresPassengerAccountRepository struct{ db *sql.DB }

func NewPostgresPassengerAccountRepository(db *sql.DB) *PostgresPassengerAccountRepository {
	return &PostgresPassengerAccountRepository{db: db}
}

// account details
func (r *PostgresPassengerAccountRepository) Create(ctx context.Context, acct models.PassengerAccount) (models.PassengerAccount, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO passenger_accounts (id, passenger_id, account_number, bank_code, bank_name, priority_order, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		acct.ID, acct.PassengerID, acct.AccountNumber, acct.BankCode, acct.BankName, acct.PriorityOrder, acct.IsActive)
	return acct, err
}

func (r *PostgresPassengerAccountRepository) CountByPassenger(ctx context.Context, passengerID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM passenger_accounts WHERE passenger_id = $1`, passengerID,
	).Scan(&count)
	return count, err
}

func (r *PostgresPassengerAccountRepository) FindActiveByPassenger(ctx context.Context, passengerID uuid.UUID) ([]models.PassengerAccount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, passenger_id, account_number, bank_code, bank_name, priority_order, is_active
		 FROM passenger_accounts WHERE passenger_id = $1 AND is_active = true ORDER BY priority_order ASC`, passengerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.PassengerAccount
	for rows.Next() {
		var acct models.PassengerAccount
		if err := rows.Scan(&acct.ID, &acct.PassengerID, &acct.AccountNumber, &acct.BankCode, &acct.BankName, &acct.PriorityOrder, &acct.IsActive); err != nil {
			return nil, err
		}
		accounts = append(accounts, acct)
	}
	return accounts, rows.Err()
}

type PostgresTerminalRepository struct{ db *sql.DB }

func NewPostgresTerminalRepository(db *sql.DB) *PostgresTerminalRepository {
	return &PostgresTerminalRepository{db: db}
}

// create-terminal()
func (r *PostgresTerminalRepository) Create(ctx context.Context, term models.Terminal) (models.Terminal, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO terminals (id, driver_id, device_serial, vehicle_plate, terminal_type, vehicle_type, last_lat, last_lng, last_seen_at, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		term.ID, term.DriverID, term.DeviceSerial, term.VehiclePlate, term.TerminalType, term.VehicleType, term.LastLat, term.LastLng, term.LastSeenAt, term.IsActive)
	return term, err
}

// findbyId-terminal()
func (r *PostgresTerminalRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Terminal, error) {
	var term models.Terminal
	err := r.db.QueryRowContext(ctx,
		`SELECT id, driver_id, device_serial, vehicle_plate, terminal_type, vehicle_type, last_lat, last_lng, last_seen_at, is_active
		 FROM terminals WHERE id = $1`, id,
	).Scan(&term.ID, &term.DriverID, &term.DeviceSerial, &term.VehiclePlate, &term.TerminalType, &term.VehicleType, &term.LastLat, &term.LastLng, &term.LastSeenAt, &term.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &term, nil
}

// findByDeviceSerial-terminal()
func (r *PostgresTerminalRepository) FindByDeviceSerial(ctx context.Context, serial string) (*models.Terminal, error) {
	var term models.Terminal
	err := r.db.QueryRowContext(ctx,
		`SELECT id, driver_id, device_serial, vehicle_plate, terminal_type, vehicle_type, last_lat, last_lng, last_seen_at, is_active
		 FROM terminals WHERE device_serial = $1`, serial,
	).Scan(&term.ID, &term.DriverID, &term.DeviceSerial, &term.VehiclePlate, &term.TerminalType, &term.VehicleType, &term.LastLat, &term.LastLng, &term.LastSeenAt, &term.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &term, nil
}

type PostgresTripRepository struct{ db *sql.DB }

func NewPostgresTripRepository(db *sql.DB) *PostgresTripRepository {
	return &PostgresTripRepository{db: db}
}

// createtrip-terminal()
func (r *PostgresTripRepository) Create(ctx context.Context, trp models.Trip) (models.Trip, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trips (id, passenger_id, driver_id, terminal_id, fare_amount, driver_payout, platform_fee, pickup_lat, pickup_lng, status, tapped_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		trp.ID, trp.PassengerID, trp.DriverID, trp.TerminalID, trp.FareAmount, trp.DriverPayout, trp.PlatformFee, trp.PickupLat, trp.PickupLng, trp.Status, trp.TappedAt)
	return trp, err
}

// findbypassenger
func (r *PostgresTripRepository) FindByPassenger(ctx context.Context, passengerID uuid.UUID) ([]models.Trip, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, passenger_id, driver_id, terminal_id,
		        fare_amount, driver_payout, platform_fee,
		        pickup_lat, pickup_lng, status, tapped_at
		 FROM trips
		 WHERE passenger_id = $1
		 ORDER BY tapped_at DESC`,
		passengerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []models.Trip

	for rows.Next() {
		var trp models.Trip

		if err := rows.Scan(
			&trp.ID,
			&trp.PassengerID,
			&trp.DriverID,
			&trp.TerminalID,
			&trp.FareAmount,
			&trp.DriverPayout,
			&trp.PlatformFee,
			&trp.PickupLat,
			&trp.PickupLng,
			&trp.Status,
			&trp.TappedAt,
		); err != nil {
			return nil, err
		}

		trips = append(trips, trp)
	}

	return trips, rows.Err()
}

// findbydriver trips()
func (r *PostgresTripRepository) FindByDriver(ctx context.Context, driverID uuid.UUID) ([]models.Trip, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, passenger_id, driver_id, terminal_id,
		        fare_amount, driver_payout, platform_fee,
		        pickup_lat, pickup_lng, status, tapped_at
		 FROM trips
		 WHERE driver_id = $1
		 ORDER BY tapped_at DESC`,
		driverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []models.Trip

	for rows.Next() {
		var trp models.Trip

		if err := rows.Scan(
			&trp.ID,
			&trp.PassengerID,
			&trp.DriverID,
			&trp.TerminalID,
			&trp.FareAmount,
			&trp.DriverPayout,
			&trp.PlatformFee,
			&trp.PickupLat,
			&trp.PickupLng,
			&trp.Status,
			&trp.TappedAt,
		); err != nil {
			return nil, err
		}

		trips = append(trips, trp)
	}

	return trips, rows.Err()
}
