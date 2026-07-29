package repository

import (
	"context"
	"database/sql"
	"errors"

	"transportation/internal/models"
)

type PostgresUserRepository struct{ db *sql.DB }

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, u models.User) (models.User, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, phone, full_name, role, created_at) VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Phone, u.FullName, u.Role, u.CreatedAt)
	return u, err
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, phone, full_name, role, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Phone, &u.FullName, &u.Role, &u.CreatedAt)
	return u, err
}

type PostgresPassengerAccountRepository struct{ db *sql.DB }

func NewPostgresPassengerAccountRepository(db *sql.DB) *PostgresPassengerAccountRepository {
	return &PostgresPassengerAccountRepository{db: db}
}

func (r *PostgresPassengerAccountRepository) Create(ctx context.Context, a models.PassengerAccount) (models.PassengerAccount, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO passenger_accounts (id, passenger_id, account_number, bank_code, bank_name, priority_order, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.ID, a.PassengerID, a.AccountNumber, a.BankCode, a.BankName, a.PriorityOrder, a.IsActive)
	return a, err
}

func (r *PostgresPassengerAccountRepository) CountByPassenger(ctx context.Context, passengerID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM passenger_accounts WHERE passenger_id = $1`, passengerID,
	).Scan(&count)
	return count, err
}

func (r *PostgresPassengerAccountRepository) FindActiveByPassenger(ctx context.Context, passengerID string) ([]models.PassengerAccount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, passenger_id, account_number, bank_code, bank_name, priority_order, is_active
		 FROM passenger_accounts WHERE passenger_id = $1 AND is_active = true ORDER BY priority_order ASC`, passengerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.PassengerAccount
	for rows.Next() {
		var a models.PassengerAccount
		if err := rows.Scan(&a.ID, &a.PassengerID, &a.AccountNumber, &a.BankCode, &a.BankName, &a.PriorityOrder, &a.IsActive); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

type PostgresTerminalRepository struct{ db *sql.DB }

func NewPostgresTerminalRepository(db *sql.DB) *PostgresTerminalRepository {
	return &PostgresTerminalRepository{db: db}
}

func (r *PostgresTerminalRepository) Create(ctx context.Context, t models.Terminal) (models.Terminal, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO terminals (id, driver_id, device_serial, vehicle_plate, terminal_type, last_lat, last_lng, last_seen_at, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		t.ID, t.DriverID, t.DeviceSerial, t.VehiclePlate, t.TerminalType, t.LastLat, t.LastLng, t.LastSeenAt, t.IsActive)
	return t, err
}

func (r *PostgresTerminalRepository) FindByID(ctx context.Context, id string) (*models.Terminal, error) {
	var t models.Terminal
	err := r.db.QueryRowContext(ctx,
		`SELECT id, driver_id, device_serial, vehicle_plate, terminal_type, last_lat, last_lng, last_seen_at, is_active
		 FROM terminals WHERE id = $1`, id,
	).Scan(&t.ID, &t.DriverID, &t.DeviceSerial, &t.VehiclePlate, &t.TerminalType, &t.LastLat, &t.LastLng, &t.LastSeenAt, &t.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PostgresTerminalRepository) FindByDeviceSerial(ctx context.Context, serial string) (*models.Terminal, error) {
	var t models.Terminal
	err := r.db.QueryRowContext(ctx,
		`SELECT id, driver_id, device_serial, vehicle_plate, terminal_type, last_lat, last_lng, last_seen_at, is_active
		 FROM terminals WHERE device_serial = $1`, serial,
	).Scan(&t.ID, &t.DriverID, &t.DeviceSerial, &t.VehiclePlate, &t.TerminalType, &t.LastLat, &t.LastLng, &t.LastSeenAt, &t.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type PostgresTripRepository struct{ db *sql.DB }

func NewPostgresTripRepository(db *sql.DB) *PostgresTripRepository {
	return &PostgresTripRepository{db: db}
}

func (r *PostgresTripRepository) Create(ctx context.Context, t models.Trip) (models.Trip, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trips (id, passenger_id, driver_id, terminal_id, fare_amount, driver_payout, platform_fee, pickup_lat, pickup_lng, status, tapped_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		t.ID, t.PassengerID, t.DriverID, t.TerminalID, t.FareAmount, t.DriverPayout, t.PlatformFee, t.PickupLat, t.PickupLng, t.Status, t.TappedAt)
	return t, err
}

func (r *PostgresTripRepository) FindByPassengerOrDriver(ctx context.Context, passengerID, driverID string) ([]models.Trip, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, passenger_id, driver_id, terminal_id, fare_amount, driver_payout, platform_fee, pickup_lat, pickup_lng, status, tapped_at
		 FROM trips WHERE ($1 <> '' AND passenger_id = $1) OR ($2 <> '' AND driver_id = $2)`, passengerID, driverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []models.Trip
	for rows.Next() {
		var t models.Trip
		if err := rows.Scan(&t.ID, &t.PassengerID, &t.DriverID, &t.TerminalID, &t.FareAmount, &t.DriverPayout, &t.PlatformFee, &t.PickupLat, &t.PickupLng, &t.Status, &t.TappedAt); err != nil {
			return nil, err
		}
		trips = append(trips, t)
	}
	return trips, rows.Err()
}
