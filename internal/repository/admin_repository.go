package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

	if err := ensureTaxiParkRolePermissions(ctx, transaction); err != nil {
		return admin.CreateTaxiParkOwnerResult{}, err
	}

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

func (repository *PostgresAdminRepository) ListTaxiParkAccounts(ctx context.Context, filter admin.ListTaxiParkAccountsFilter) ([]admin.TaxiParkAccount, error) {
	const query = `
		SELECT
			tp.id,
			tp.owner_user_id,
			tp.city_id,
			c.name,
			tp.name,
			COALESCE(tp.legal_name, ''),
			COALESCE(tp.tax_id, ''),
			tp.contact_phone,
			tp.contact_email,
			u.phone,
			COALESCE(u.email, ''),
			tp.is_verified,
			tp.verification_status::text,
			COALESCE(tp.commission_percent::text, ''),
			COALESCE(tp.balance_cents, 0),
			u.is_active,
			tp.created_at,
			tp.deleted_at
		FROM taxi_parks tp
		JOIN users u ON u.id = tp.owner_user_id
		JOIN cities c ON c.id = tp.city_id
		WHERE ($2::boolean OR tp.deleted_at IS NULL)
		  AND (
			$3 = ''
			OR tp.name ILIKE '%' || $3 || '%'
			OR COALESCE(tp.legal_name, '') ILIKE '%' || $3 || '%'
			OR COALESCE(tp.tax_id, '') ILIKE '%' || $3 || '%'
			OR tp.contact_phone ILIKE '%' || $3 || '%'
			OR tp.contact_email ILIKE '%' || $3 || '%'
			OR u.phone ILIKE '%' || $3 || '%'
			OR COALESCE(u.email, '') ILIKE '%' || $3 || '%'
		  )
		ORDER BY tp.created_at DESC
		LIMIT $1`

	rows, err := repository.pool.Query(ctx, query, filter.Limit, filter.IncludeDeleted, filter.Search)
	if err != nil {
		return nil, fmt.Errorf("query taxi park accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]admin.TaxiParkAccount, 0, filter.Limit)
	for rows.Next() {
		var account admin.TaxiParkAccount
		var deletedAt pgtype.Timestamptz
		if err := rows.Scan(
			&account.TaxiParkID,
			&account.OwnerUserID,
			&account.CityID,
			&account.CityName,
			&account.Name,
			&account.LegalName,
			&account.TaxID,
			&account.ContactPhone,
			&account.ContactEmail,
			&account.OwnerPhone,
			&account.OwnerEmail,
			&account.IsVerified,
			&account.VerificationStatus,
			&account.CommissionPercent,
			&account.BalanceCents,
			&account.IsOwnerActive,
			&account.CreatedAt,
			&deletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan taxi park account: %w", err)
		}
		if deletedAt.Valid {
			account.DeletedAt = &deletedAt.Time
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park accounts: %w", err)
	}

	return accounts, nil
}

func ensureTaxiParkRolePermissions(ctx context.Context, transaction pgx.Tx) error {
	if _, err := transaction.Exec(ctx, `
		INSERT INTO permissions (code, description)
		VALUES
			('taxi_park.profile.manage', 'Taxi park can manage own profile'),
			('taxi_park.drivers.create', 'Taxi park can create own drivers'),
			('taxi_park.drivers.manage', 'Taxi park can manage own drivers'),
			('taxi_park.cars.manage', 'Taxi park can manage own cars'),
			('taxi_park.orders.view', 'Taxi park can view own fleet orders'),
			('taxi_park.earnings.view', 'Taxi park can view own fleet earnings'),
			('taxi_park.finance.view', 'Taxi park can view own finance')
		ON CONFLICT (code) DO NOTHING`); err != nil {
		return fmt.Errorf("ensure taxi park permissions: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO role_permissions (role, permission_code)
		VALUES
			('taxi_park', 'taxi_park.profile.manage'),
			('taxi_park', 'taxi_park.drivers.create'),
			('taxi_park', 'taxi_park.drivers.manage'),
			('taxi_park', 'taxi_park.cars.manage'),
			('taxi_park', 'taxi_park.orders.view'),
			('taxi_park', 'taxi_park.earnings.view'),
			('taxi_park', 'taxi_park.finance.view')
		ON CONFLICT (role, permission_code) DO NOTHING`); err != nil {
		return fmt.Errorf("ensure taxi park role permissions: %w", err)
	}

	return nil
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
