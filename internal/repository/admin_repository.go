package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/admin"
	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresAdminRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAdminRepository(pool *pgxpool.Pool) *PostgresAdminRepository {
	return &PostgresAdminRepository{pool: pool}
}

func (repository *PostgresAdminRepository) CreateTaxiParkOwner(ctx context.Context, record admin.CreateTaxiParkOwnerRecord) (admin.CreateTaxiParkOwnerResult, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return admin.CreateTaxiParkOwnerResult{}, fmt.Errorf("begin create taxi park transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	const insertUserQuery = `
		INSERT INTO users (
			phone,
			email,
			role,
			registration_type,
			first_name,
			last_name,
			password_hash,
			is_phone_confirmed,
			phone_confirmed_at,
			is_email_confirmed,
			email_confirmed_at,
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
		VALUES (
			$1, $2, 'taxi_park', 'taxi_park', $3, $4, $5,
			true, now(), true, now(), true, now(), $6, true, now(), $7, $8, $9, true
		)
		RETURNING id`

	var result admin.CreateTaxiParkOwnerResult
	if err := transaction.QueryRow(
		ctx,
		insertUserQuery,
		record.Phone,
		record.Email,
		nullableString(record.FirstName),
		nullableString(record.LastName),
		record.PasswordHash,
		record.PrivacyPolicyVersion,
		record.TermsVersion,
		nullableString(record.ConsentIP),
		nullableString(record.ConsentUserAgent),
	).Scan(&result.UserID); err != nil {
		return admin.CreateTaxiParkOwnerResult{}, fmt.Errorf("insert taxi park owner user: %w", err)
	}

	if err := insertConsoleConsentEvents(ctx, transaction, result.UserID, record); err != nil {
		return admin.CreateTaxiParkOwnerResult{}, err
	}

	const insertTaxiParkQuery = `
		INSERT INTO taxi_parks (
			owner_user_id,
			city_id,
			name,
			legal_name,
			tax_id,
			contact_phone,
			contact_email,
			is_verified,
			verification_status,
			commission_percent
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			CASE WHEN $8 THEN 'verified'::verification_status ELSE 'pending'::verification_status END,
			$9
		)
		RETURNING id`

	if err := transaction.QueryRow(
		ctx,
		insertTaxiParkQuery,
		result.UserID,
		record.CityID,
		record.Name,
		nullableString(record.LegalName),
		nullableString(record.TaxID),
		record.Phone,
		record.Email,
		record.Verified,
		record.CommissionPercent,
	).Scan(&result.TaxiParkID); err != nil {
		return admin.CreateTaxiParkOwnerResult{}, fmt.Errorf("insert taxi park: %w", err)
	}

	const insertSettingsQuery = `
		INSERT INTO taxi_park_settings (
			taxi_park_id,
			display_name,
			short_name,
			support_phone,
			support_email,
			legal_name,
			inn,
			commission_percent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (taxi_park_id) DO NOTHING`
	if _, err := transaction.Exec(
		ctx,
		insertSettingsQuery,
		result.TaxiParkID,
		record.Name,
		nullableString(record.Name),
		record.Phone,
		record.Email,
		nullableString(record.LegalName),
		nullableString(record.TaxID),
		record.CommissionPercent,
	); err != nil {
		return admin.CreateTaxiParkOwnerResult{}, fmt.Errorf("insert taxi park settings: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return admin.CreateTaxiParkOwnerResult{}, fmt.Errorf("commit create taxi park transaction: %w", err)
	}

	result.Phone = record.Phone
	result.Email = record.Email
	return result, nil
}

func (repository *PostgresAdminRepository) ResetPasswordByPhone(ctx context.Context, record admin.ResetPasswordRecord) (admin.ResetPasswordResult, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return admin.ResetPasswordResult{}, fmt.Errorf("begin reset password transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	const updateUserQuery = `
		UPDATE users
		SET password_hash = $3,
		    updated_at = now()
		WHERE phone = $1
		  AND role = $2
		  AND deleted_at IS NULL
		RETURNING id`

	result := admin.ResetPasswordResult{
		Phone: record.Phone,
		Role:  record.Role,
	}
	if err := transaction.QueryRow(ctx, updateUserQuery, record.Phone, record.Role, record.PasswordHash).Scan(&result.UserID); err != nil {
		return admin.ResetPasswordResult{}, fmt.Errorf("update user password hash: %w", err)
	}

	commandTag, err := transaction.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`, result.UserID)
	if err != nil {
		return admin.ResetPasswordResult{}, fmt.Errorf("revoke refresh tokens after password reset: %w", err)
	}
	result.RevokedTokenCount = commandTag.RowsAffected()

	if err := transaction.Commit(ctx); err != nil {
		return admin.ResetPasswordResult{}, fmt.Errorf("commit reset password transaction: %w", err)
	}

	return result, nil
}

func insertConsoleConsentEvents(ctx context.Context, transaction pgx.Tx, userID uuid.UUID, record admin.CreateTaxiParkOwnerRecord) error {
	events := []struct {
		documentType    domain.ConsentDocumentType
		documentVersion string
	}{
		{documentType: domain.ConsentDocumentPersonalData, documentVersion: record.PrivacyPolicyVersion},
		{documentType: domain.ConsentDocumentPrivacyPolicy, documentVersion: record.PrivacyPolicyVersion},
		{documentType: domain.ConsentDocumentTerms, documentVersion: record.TermsVersion},
	}

	for _, event := range events {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO user_consent_events (
				user_id,
				event_type,
				document_type,
				document_version,
				ip,
				user_agent
			)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			userID,
			domain.ConsentEventAccepted,
			event.documentType,
			event.documentVersion,
			nullableString(record.ConsentIP),
			nullableString(record.ConsentUserAgent),
		); err != nil {
			return fmt.Errorf("insert console consent event: %w", err)
		}
	}

	return nil
}
