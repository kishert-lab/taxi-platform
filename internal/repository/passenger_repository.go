package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresPassengerRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPassengerRepository(pool *pgxpool.Pool) *PostgresPassengerRepository {
	return &PostgresPassengerRepository{pool: pool}
}

func (repository *PostgresPassengerRepository) Create(ctx context.Context, passenger domain.Passenger) (domain.Passenger, error) {
	const query = `
		INSERT INTO passengers (phone, name, email, avatar_url, is_active, phone_verified_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + passengerSelectColumns

	record, err := scanPassenger(repository.pool.QueryRow(
		ctx,
		query,
		passenger.Phone,
		nullableString(passenger.Name),
		nullableString(passenger.Email),
		nullableString(passenger.AvatarURL),
		passenger.IsActive,
		passenger.PhoneVerifiedAt,
		passenger.LastLoginAt,
	))
	if err != nil {
		return domain.Passenger{}, fmt.Errorf("insert passenger: %w", err)
	}

	return record, nil
}

func (repository *PostgresPassengerRepository) GetByID(ctx context.Context, passengerID uuid.UUID) (domain.Passenger, error) {
	const query = `SELECT ` + passengerSelectColumns + ` FROM passengers WHERE id = $1 AND deleted_at IS NULL`

	record, err := scanPassenger(repository.pool.QueryRow(ctx, query, passengerID))
	if err != nil {
		return domain.Passenger{}, fmt.Errorf("select passenger by id: %w", err)
	}

	return record, nil
}

func (repository *PostgresPassengerRepository) GetByPhone(ctx context.Context, phone string) (domain.Passenger, error) {
	const query = `SELECT ` + passengerSelectColumns + ` FROM passengers WHERE phone = $1 AND deleted_at IS NULL`

	record, err := scanPassenger(repository.pool.QueryRow(ctx, query, phone))
	if err != nil {
		return domain.Passenger{}, fmt.Errorf("select passenger by phone: %w", err)
	}

	return record, nil
}

func (repository *PostgresPassengerRepository) UpdateProfile(ctx context.Context, passengerID uuid.UUID, name string, email string, avatarURL string) (domain.Passenger, error) {
	const query = `
		UPDATE passengers
		SET name = $2,
		    email = $3,
		    avatar_url = $4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + passengerSelectColumns

	record, err := scanPassenger(repository.pool.QueryRow(
		ctx,
		query,
		passengerID,
		nullableString(name),
		nullableString(email),
		nullableString(avatarURL),
	))
	if err != nil {
		return domain.Passenger{}, fmt.Errorf("update passenger profile: %w", err)
	}

	return record, nil
}

func (repository *PostgresPassengerRepository) MarkAuthenticated(ctx context.Context, passengerID uuid.UUID, phoneVerifiedAt *time.Time, lastLoginAt time.Time) (domain.Passenger, error) {
	const query = `
		UPDATE passengers
		SET phone_verified_at = COALESCE($2, phone_verified_at),
		    last_login_at = $3
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + passengerSelectColumns

	record, err := scanPassenger(repository.pool.QueryRow(ctx, query, passengerID, phoneVerifiedAt, lastLoginAt))
	if err != nil {
		return domain.Passenger{}, fmt.Errorf("mark passenger authenticated: %w", err)
	}

	return record, nil
}

const passengerSelectColumns = `
	id,
	phone,
	name,
	email,
	avatar_url,
	is_active,
	phone_verified_at,
	last_login_at,
	created_at,
	updated_at,
	deleted_at`

func scanPassenger(row pgx.Row) (domain.Passenger, error) {
	var passenger domain.Passenger
	var name pgtype.Text
	var email pgtype.Text
	var avatarURL pgtype.Text
	var phoneVerifiedAt pgtype.Timestamptz
	var lastLoginAt pgtype.Timestamptz
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&passenger.ID,
		&passenger.Phone,
		&name,
		&email,
		&avatarURL,
		&passenger.IsActive,
		&phoneVerifiedAt,
		&lastLoginAt,
		&passenger.CreatedAt,
		&passenger.UpdatedAt,
		&deletedAt,
	); err != nil {
		return domain.Passenger{}, err
	}

	passenger.Name = name.String
	passenger.Email = email.String
	passenger.AvatarURL = avatarURL.String
	if phoneVerifiedAt.Valid {
		passenger.PhoneVerifiedAt = &phoneVerifiedAt.Time
	}
	if lastLoginAt.Valid {
		passenger.LastLoginAt = &lastLoginAt.Time
	}
	if deletedAt.Valid {
		passenger.DeletedAt = &deletedAt.Time
	}

	return passenger, nil
}
