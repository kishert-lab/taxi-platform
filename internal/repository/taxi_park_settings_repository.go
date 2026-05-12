package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type PostgresTaxiParkSettingsRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTaxiParkSettingsRepository(pool *pgxpool.Pool) *PostgresTaxiParkSettingsRepository {
	return &PostgresTaxiParkSettingsRepository{pool: pool}
}

func (repository *PostgresTaxiParkSettingsRepository) GetSettingsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error) {
	if err := repository.ensureSettings(ctx, ownerUserID); err != nil {
		return domain.TaxiParkSettings{}, err
	}
	settings, err := scanTaxiParkSettings(repository.pool.QueryRow(ctx, `SELECT `+taxiParkSettingsColumns+`
		FROM taxi_park_settings s
		JOIN taxi_parks p ON p.id = s.taxi_park_id
		WHERE p.owner_user_id = $1 AND p.deleted_at IS NULL`, ownerUserID))
	if err != nil {
		return domain.TaxiParkSettings{}, fmt.Errorf("select taxi park settings: %w", err)
	}
	return settings, nil
}

func (repository *PostgresTaxiParkSettingsRepository) UpdateSettingsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error) {
	if err := repository.ensureSettings(ctx, ownerUserID); err != nil {
		return domain.TaxiParkSettings{}, err
	}

	settings, err := scanTaxiParkSettings(repository.pool.QueryRow(ctx, `
		UPDATE taxi_park_settings s
		SET display_name = COALESCE($2, display_name),
		    short_name = COALESCE($3, short_name),
		    support_phone = COALESCE($4, support_phone),
		    support_email = COALESCE($5, support_email),
		    legal_name = COALESCE($6, legal_name),
		    legal_address = COALESCE($7, legal_address),
		    inn = COALESCE($8, inn),
		    ogrn = COALESCE($9, ogrn),
		    website = COALESCE($10, website),
		    logo_url = COALESCE($11, logo_url),
		    primary_color = COALESCE($12, primary_color),
		    secondary_color = COALESCE($13, secondary_color),
		    commission_percent = COALESCE($14::numeric / 100, commission_percent),
		    minimum_order_price_cents = COALESCE($15, minimum_order_price_cents),
		    cancellation_timeout_sec = COALESCE($16, cancellation_timeout_sec),
		    driver_arrival_timeout_sec = COALESCE($17, driver_arrival_timeout_sec),
		    allow_cash_payment = COALESCE($18, allow_cash_payment),
		    allow_card_payment = COALESCE($19, allow_card_payment),
		    allow_transfer_payment = COALESCE($20, allow_transfer_payment),
		    is_active = COALESCE($21, is_active)
		FROM taxi_parks p
		WHERE p.id = s.taxi_park_id AND p.owner_user_id = $1 AND p.deleted_at IS NULL
		RETURNING `+taxiParkSettingsColumns,
		ownerUserID,
		request.DisplayName,
		request.ShortName,
		request.SupportPhone,
		request.SupportEmail,
		request.LegalName,
		request.LegalAddress,
		request.INN,
		request.OGRN,
		request.Website,
		request.LogoURL,
		request.PrimaryColor,
		request.SecondaryColor,
		request.CommissionBasisPoints,
		request.MinimumOrderPriceCents,
		request.CancellationTimeoutSec,
		request.DriverArrivalTimeoutSec,
		request.AllowCashPayment,
		request.AllowCardPayment,
		request.AllowTransferPayment,
		request.IsActive,
	))
	if err != nil {
		return domain.TaxiParkSettings{}, fmt.Errorf("update taxi park settings: %w", err)
	}
	return settings, nil
}

func (repository *PostgresTaxiParkSettingsRepository) ListTariffsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) ([]domain.TaxiParkTariff, error) {
	rows, err := repository.pool.Query(ctx, `SELECT `+taxiParkTariffSelectColumns+`
		FROM taxi_park_tariffs t
		JOIN taxi_parks p ON p.id = t.taxi_park_id
		WHERE p.owner_user_id = $1 AND p.deleted_at IS NULL
		ORDER BY t.created_at DESC`, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("select taxi park tariffs: %w", err)
	}
	defer rows.Close()
	return scanTaxiParkTariffs(rows)
}

func (repository *PostgresTaxiParkSettingsRepository) CreateTariffByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error) {
	fixedRoutes := request.FixedRoutes
	if len(fixedRoutes) == 0 {
		fixedRoutes = []byte("[]")
	}
	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}

	tariff, err := scanTaxiParkTariff(repository.pool.QueryRow(ctx, `
		INSERT INTO taxi_park_tariffs (
			taxi_park_id, name, description, base_price_cents, price_per_km_cents,
			price_per_minute_cents, minimum_price_cents, fixed_routes, is_active
		)
		SELECT id, $2, $3, $4, $5, $6, $7, $8, $9
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL
		RETURNING `+taxiParkTariffReturnColumns,
		ownerUserID,
		request.Name,
		nullableString(request.Description),
		request.BasePriceCents,
		request.PricePerKMCents,
		request.PricePerMinuteCents,
		request.MinimumPriceCents,
		fixedRoutes,
		isActive,
	))
	if err != nil {
		return domain.TaxiParkTariff{}, fmt.Errorf("insert taxi park tariff: %w", err)
	}
	return tariff, nil
}

func (repository *PostgresTaxiParkSettingsRepository) UpdateTariffByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, tariffID uuid.UUID, request dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error) {
	var fixedRoutes any
	if len(request.FixedRoutes) > 0 {
		fixedRoutes = request.FixedRoutes
	}
	tariff, err := scanTaxiParkTariff(repository.pool.QueryRow(ctx, `
		UPDATE taxi_park_tariffs t
		SET name = COALESCE($3, name),
		    description = COALESCE($4, description),
		    base_price_cents = COALESCE($5, base_price_cents),
		    price_per_km_cents = COALESCE($6, price_per_km_cents),
		    price_per_minute_cents = COALESCE($7, price_per_minute_cents),
		    minimum_price_cents = COALESCE($8, minimum_price_cents),
		    fixed_routes = COALESCE($9, fixed_routes),
		    is_active = COALESCE($10, is_active)
		FROM taxi_parks p
		WHERE p.id = t.taxi_park_id
		  AND p.owner_user_id = $1
		  AND t.id = $2
		  AND p.deleted_at IS NULL
		RETURNING `+taxiParkTariffReturnColumns,
		ownerUserID,
		tariffID,
		request.Name,
		request.Description,
		request.BasePriceCents,
		request.PricePerKMCents,
		request.PricePerMinuteCents,
		request.MinimumPriceCents,
		fixedRoutes,
		request.IsActive,
	))
	if err != nil {
		return domain.TaxiParkTariff{}, fmt.Errorf("update taxi park tariff: %w", err)
	}
	return tariff, nil
}

func (repository *PostgresTaxiParkSettingsRepository) ensureSettings(ctx context.Context, ownerUserID uuid.UUID) error {
	const query = `
		INSERT INTO taxi_park_settings (taxi_park_id, display_name, short_name, support_phone, support_email, legal_name)
		SELECT id, name, name, contact_phone, contact_email, legal_name
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL
		ON CONFLICT (taxi_park_id) DO NOTHING`
	if _, err := repository.pool.Exec(ctx, query, ownerUserID); err != nil {
		return fmt.Errorf("ensure taxi park settings: %w", err)
	}
	return nil
}

const taxiParkSettingsColumns = `
	s.id, s.taxi_park_id, s.display_name, COALESCE(s.short_name, ''), COALESCE(s.support_phone, ''),
	COALESCE(s.support_email, ''), COALESCE(s.legal_name, ''), COALESCE(s.legal_address, ''),
	COALESCE(s.inn, ''), COALESCE(s.ogrn, ''), COALESCE(s.website, ''), COALESCE(s.logo_url, ''),
	COALESCE(s.primary_color, ''), COALESCE(s.secondary_color, ''), (s.commission_percent * 100)::integer,
	s.minimum_order_price_cents, s.cancellation_timeout_sec, s.driver_arrival_timeout_sec,
	s.allow_cash_payment, s.allow_card_payment, s.allow_transfer_payment, s.is_active,
	s.created_at, s.updated_at`

func scanTaxiParkSettings(row pgx.Row) (domain.TaxiParkSettings, error) {
	var settings domain.TaxiParkSettings
	var commissionBasisPoints pgtype.Int4
	if err := row.Scan(
		&settings.ID,
		&settings.TaxiParkID,
		&settings.DisplayName,
		&settings.ShortName,
		&settings.SupportPhone,
		&settings.SupportEmail,
		&settings.LegalName,
		&settings.LegalAddress,
		&settings.INN,
		&settings.OGRN,
		&settings.Website,
		&settings.LogoURL,
		&settings.PrimaryColor,
		&settings.SecondaryColor,
		&commissionBasisPoints,
		&settings.MinimumOrderPrice.Amount,
		&settings.CancellationTimeoutSec,
		&settings.DriverArrivalTimeoutSec,
		&settings.AllowCashPayment,
		&settings.AllowCardPayment,
		&settings.AllowTransferPayment,
		&settings.IsActive,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	); err != nil {
		return domain.TaxiParkSettings{}, err
	}
	settings.MinimumOrderPrice.Currency = "RUB"
	if commissionBasisPoints.Valid {
		value := commissionBasisPoints.Int32
		settings.CommissionBasisPoints = &value
	}
	return settings, nil
}

const taxiParkTariffSelectColumns = `
	t.id, t.taxi_park_id, t.name, COALESCE(t.description, ''),
	t.base_price_cents, t.price_per_km_cents, t.price_per_minute_cents,
	t.minimum_price_cents, t.fixed_routes, t.is_active, t.created_at, t.updated_at`

const taxiParkTariffReturnColumns = `
	id, taxi_park_id, name, COALESCE(description, ''),
	base_price_cents, price_per_km_cents, price_per_minute_cents,
	minimum_price_cents, fixed_routes, is_active, created_at, updated_at`

func scanTaxiParkTariffs(rows pgx.Rows) ([]domain.TaxiParkTariff, error) {
	tariffs := make([]domain.TaxiParkTariff, 0)
	for rows.Next() {
		tariff, err := scanTaxiParkTariff(rows)
		if err != nil {
			return nil, err
		}
		tariffs = append(tariffs, tariff)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park tariffs: %w", err)
	}
	return tariffs, nil
}

func scanTaxiParkTariff(row pgx.Row) (domain.TaxiParkTariff, error) {
	var tariff domain.TaxiParkTariff
	var fixedRoutes []byte
	if err := row.Scan(
		&tariff.ID,
		&tariff.TaxiParkID,
		&tariff.Name,
		&tariff.Description,
		&tariff.BasePrice.Amount,
		&tariff.PricePerKM.Amount,
		&tariff.PricePerMinute.Amount,
		&tariff.MinimumPrice.Amount,
		&fixedRoutes,
		&tariff.IsActive,
		&tariff.CreatedAt,
		&tariff.UpdatedAt,
	); err != nil {
		return domain.TaxiParkTariff{}, fmt.Errorf("scan taxi park tariff: %w", err)
	}
	if !json.Valid(fixedRoutes) {
		fixedRoutes = []byte("[]")
	}
	tariff.FixedRoutes = fixedRoutes
	tariff.BasePrice.Currency = "RUB"
	tariff.PricePerKM.Currency = "RUB"
	tariff.PricePerMinute.Currency = "RUB"
	tariff.MinimumPrice.Currency = "RUB"
	return tariff, nil
}
