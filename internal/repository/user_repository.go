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

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (repository *PostgresUserRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	const query = `
		INSERT INTO users (
			phone,
			email,
			role,
			registration_type,
			first_name,
			last_name,
			password_hash,
			is_phone_confirmed,
			is_email_confirmed,
			personal_data_consent,
			personal_data_consent_at,
			privacy_policy_version,
			terms_accepted,
			terms_accepted_at,
			terms_version,
			consent_ip,
			consent_user_agent,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING ` + userSelectColumns

	createdUser, err := scanUser(repository.pool.QueryRow(
		ctx,
		query,
		user.Phone,
		user.Email,
		user.Role,
		user.RegistrationType,
		nullableString(user.FirstName),
		nullableString(user.LastName),
		user.PasswordHash,
		user.IsPhoneConfirmed,
		user.IsEmailConfirmed,
		user.PersonalDataConsent,
		user.PersonalDataConsentAt,
		nullableString(user.PrivacyPolicyVersion),
		user.TermsAccepted,
		user.TermsAcceptedAt,
		nullableString(user.TermsVersion),
		nullableString(user.ConsentIP),
		nullableString(user.ConsentUserAgent),
		user.IsActive,
	))
	if err != nil {
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	return createdUser, nil
}

func (repository *PostgresUserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	const query = `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1 AND deleted_at IS NULL`

	user, err := scanUser(repository.pool.QueryRow(ctx, query, userID))
	if err != nil {
		return domain.User{}, fmt.Errorf("select user by id: %w", err)
	}

	return user, nil
}

func (repository *PostgresUserRepository) GetUserByPhoneAndRole(ctx context.Context, phone string, role domain.UserRole) (domain.User, error) {
	const query = `SELECT ` + userSelectColumns + ` FROM users WHERE phone = $1 AND role = $2 AND deleted_at IS NULL`

	user, err := scanUser(repository.pool.QueryRow(ctx, query, phone, role))
	if err != nil {
		return domain.User{}, fmt.Errorf("select user by phone and role: %w", err)
	}

	return user, nil
}

func (repository *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `SELECT ` + userSelectColumns + ` FROM users WHERE lower(email) = lower($1) AND deleted_at IS NULL`

	user, err := scanUser(repository.pool.QueryRow(ctx, query, email))
	if err != nil {
		return domain.User{}, fmt.Errorf("select user by email: %w", err)
	}

	return user, nil
}

func (repository *PostgresUserRepository) MarkPhoneConfirmed(ctx context.Context, userID uuid.UUID, confirmedAt time.Time) error {
	const query = `
		UPDATE users
		SET is_phone_confirmed = true,
		    phone_confirmed_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, userID, confirmedAt)
	if err != nil {
		return fmt.Errorf("update phone confirmation: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("update phone confirmation: %w", pgx.ErrNoRows)
	}

	return nil
}

func (repository *PostgresUserRepository) MarkEmailConfirmed(ctx context.Context, userID uuid.UUID, confirmedAt time.Time) error {
	const query = `
		UPDATE users
		SET is_email_confirmed = true,
		    email_confirmed_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, userID, confirmedAt)
	if err != nil {
		return fmt.Errorf("update email confirmation: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("update email confirmation: %w", pgx.ErrNoRows)
	}

	return nil
}

const userSelectColumns = `
	id,
	phone,
	email,
	role,
	registration_type,
	first_name,
	last_name,
	profile_photo_url,
	rating::float8,
	ratings_count,
	password_hash,
	is_phone_confirmed,
	is_email_confirmed,
	personal_data_consent,
	personal_data_consent_at,
	privacy_policy_version,
	terms_accepted,
	terms_accepted_at,
	terms_version,
	consent_ip,
	consent_user_agent,
	is_active,
	last_login_at,
	created_at,
	updated_at,
	deleted_at`

func scanUser(row pgx.Row) (domain.User, error) {
	var user domain.User
	var email pgtype.Text
	var firstName pgtype.Text
	var lastName pgtype.Text
	var profilePhotoURL pgtype.Text
	var passwordHash pgtype.Text
	var personalDataConsentAt pgtype.Timestamptz
	var privacyPolicyVersion pgtype.Text
	var termsAcceptedAt pgtype.Timestamptz
	var termsVersion pgtype.Text
	var consentIP pgtype.Text
	var consentUserAgent pgtype.Text
	var lastLoginAt pgtype.Timestamptz
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&user.ID,
		&user.Phone,
		&email,
		&user.Role,
		&user.RegistrationType,
		&firstName,
		&lastName,
		&profilePhotoURL,
		&user.Rating,
		&user.RatingsCount,
		&passwordHash,
		&user.IsPhoneConfirmed,
		&user.IsEmailConfirmed,
		&user.PersonalDataConsent,
		&personalDataConsentAt,
		&privacyPolicyVersion,
		&user.TermsAccepted,
		&termsAcceptedAt,
		&termsVersion,
		&consentIP,
		&consentUserAgent,
		&user.IsActive,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&deletedAt,
	); err != nil {
		return domain.User{}, err
	}

	user.Email = email.String
	user.FirstName = firstName.String
	user.LastName = lastName.String
	user.ProfilePhotoURL = profilePhotoURL.String
	user.PasswordHash = passwordHash.String
	user.PrivacyPolicyVersion = privacyPolicyVersion.String
	user.TermsVersion = termsVersion.String
	user.ConsentIP = consentIP.String
	user.ConsentUserAgent = consentUserAgent.String
	if personalDataConsentAt.Valid {
		user.PersonalDataConsentAt = &personalDataConsentAt.Time
	}
	if termsAcceptedAt.Valid {
		user.TermsAcceptedAt = &termsAcceptedAt.Time
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Time
	}

	return user, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
