package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	taxiparkapp "github.com/kishert-lab/taxi-platform/internal/taxipark"
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

func (repository *PostgresTaxiParkSettingsRepository) CreateOrderByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record taxiparkapp.CreateOrderRecord) (domain.Order, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Order{}, fmt.Errorf("begin taxi park order transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	var taxiParkID uuid.UUID
	var cityID uuid.UUID
	if err := transaction.QueryRow(ctx, `
		SELECT id, city_id
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL`, ownerUserID).Scan(&taxiParkID, &cityID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, taxiparkapp.ErrTaxiParkNotFound
		}
		return domain.Order{}, fmt.Errorf("select taxi park for order creation: %w", err)
	}

	passengerID := ownerUserID
	if record.PassengerPhone != "" {
		firstName, lastName := splitPassengerName(record.PassengerName)
		if err := transaction.QueryRow(ctx, `
			INSERT INTO users (
				phone, role, registration_type, first_name, last_name,
				is_phone_confirmed, is_active
			)
			VALUES ($1, 'passenger', 'passenger', NULLIF($2, ''), NULLIF($3, ''), true, true)
			ON CONFLICT (phone, role) DO UPDATE
			SET first_name = COALESCE(NULLIF(EXCLUDED.first_name, ''), users.first_name),
			    last_name = COALESCE(NULLIF(EXCLUDED.last_name, ''), users.last_name),
			    updated_at = now()
			RETURNING id`, record.PassengerPhone, firstName, lastName).Scan(&passengerID); err != nil {
			return domain.Order{}, fmt.Errorf("upsert passenger for taxi park order: %w", err)
		}
	}

	metadata, err := json.Marshal(map[string]any{
		"created_by_user_id": ownerUserID,
		"created_by_role":    string(domain.UserRoleTaxiPark),
		"taxi_park_id":       taxiParkID,
		"passenger_phone":    record.PassengerPhone,
		"passenger_name":     record.PassengerName,
	})
	if err != nil {
		return domain.Order{}, fmt.Errorf("marshal taxi park order metadata: %w", err)
	}

	var destinationLatitude any
	var destinationLongitude any
	if record.DestinationLocation != nil {
		destinationLatitude = record.DestinationLocation.Latitude
		destinationLongitude = record.DestinationLocation.Longitude
	}

	orderTariffID, taxiParkTariffID, basePriceCents, pricePerKMCents, minimumPriceCents, err := repository.resolveOrderTariff(ctx, transaction, taxiParkID, record.TariffID)
	if err != nil {
		return domain.Order{}, err
	}

	order, err := scanDispatchOrder(transaction.QueryRow(ctx, `
		WITH order_points AS (
			SELECT ST_SetSRID(ST_MakePoint($6, $5), 4326)::geography AS pickup_location,
			       CASE
			           WHEN $8::double precision IS NULL OR $9::double precision IS NULL THEN NULL
			           ELSE ST_SetSRID(ST_MakePoint($9, $8), 4326)::geography
			       END AS destination_location
		),
		inserted_order AS (
			INSERT INTO orders (
				passenger_id, city_id, tariff_id, status,
				pickup_address, pickup_location,
				destination_address, destination_location,
				estimated_price, payment_method, passenger_comment,
				dispatch_attempt, version, metadata
			)
			SELECT $1, $2, $3::uuid, 'searching',
			       $4,
			       p.pickup_location,
			       $7,
			       p.destination_location,
			       CASE
			           WHEN p.destination_location IS NULL THEN (GREATEST($16::bigint, $14::bigint)::numeric / 100)
			           ELSE GREATEST(
			               $16::bigint::numeric,
			               $14::bigint::numeric + ((ST_Distance(p.pickup_location, p.destination_location) / 1000.0)::numeric * $15::bigint::numeric)
			           ) / 100
			       END,
			       $10, NULLIF($11, ''), 0, 1,
			       $12::jsonb || jsonb_build_object('taxi_park_tariff_id', $13::uuid)
			FROM order_points p
			RETURNING `+dispatchOrderSelectColumns+`
		)
		SELECT * FROM inserted_order`,
		passengerID,
		cityID,
		nullableUUID(orderTariffID),
		record.PickupAddress,
		record.PickupLocation.Latitude,
		record.PickupLocation.Longitude,
		record.DestinationAddress,
		destinationLatitude,
		destinationLongitude,
		record.PaymentMethod,
		record.Comment,
		string(metadata),
		nullableUUID(taxiParkTariffID),
		basePriceCents,
		pricePerKMCents,
		minimumPriceCents,
	))
	if err != nil {
		return domain.Order{}, fmt.Errorf("insert taxi park order: %w", err)
	}

	createdPayload, err := json.Marshal(map[string]any{
		"order_id": order.ID,
		"status":   domain.OrderStatusCreated,
	})
	if err != nil {
		return domain.Order{}, fmt.Errorf("marshal taxi park order created event: %w", err)
	}
	searchingPayload, err := json.Marshal(map[string]any{
		"order_id": order.ID,
		"status":   domain.OrderStatusSearching,
		"version":  order.Version,
	})
	if err != nil {
		return domain.Order{}, fmt.Errorf("marshal taxi park order searching event: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO order_events (order_id, actor_user_id, event_type, payload)
		VALUES ($1, $2, 'order.created', $3::jsonb),
		       ($1, $2, 'order.searching', $4::jsonb)`,
		order.ID,
		ownerUserID,
		string(createdPayload),
		string(searchingPayload),
	); err != nil {
		return domain.Order{}, fmt.Errorf("insert taxi park order events: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("commit taxi park order transaction: %w", err)
	}

	return order, nil
}

func (repository *PostgresTaxiParkSettingsRepository) CreateDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record taxiparkapp.CreateDriverRecord) (taxiparkapp.CreateDriverResult, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("begin create taxi park driver transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	var taxiParkID uuid.UUID
	var cityID uuid.UUID
	if err := transaction.QueryRow(ctx, `
		SELECT id, city_id
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL`, ownerUserID).Scan(&taxiParkID, &cityID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return taxiparkapp.CreateDriverResult{}, taxiparkapp.ErrTaxiParkNotFound
		}
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("select taxi park for driver creation: %w", err)
	}

	result := taxiparkapp.CreateDriverResult{
		TaxiParkID:                    taxiParkID,
		Phone:                         record.Phone,
		Email:                         record.Email,
		FirstName:                     record.FirstName,
		LastName:                      record.LastName,
		Status:                        domain.DriverStatusOffline,
		VerificationStatus:            record.VerificationStatus,
		Rating:                        5,
		RatingsCount:                  0,
		BirthDate:                     record.BirthDate,
		LicenseSeries:                 record.LicenseSeries,
		LicenseNumber:                 record.LicenseNumber,
		LicenseCategory:               record.LicenseCategory,
		LicenseIssuedAt:               record.LicenseIssuedAt,
		LicenseExpiresAt:              record.LicenseExpiresAt,
		DrivingExperienceFrom:         record.DrivingExperienceFrom,
		HasNoTaxiWorkRestrictions:     record.HasNoTaxiWorkRestrictions,
		FederalLaw580Compliant:        record.FederalLaw580Compliant,
		RegionalRequirementsCompliant: record.RegionalRequirementsCompliant,
		MedicalCheckPassed:            record.MedicalCheckPassed,
		PretripControlRequired:        record.PretripControlRequired,
		PretripControlPassed:          record.PretripControlPassed,
		NoTransportBan:                record.NoTransportBan,
		IsVerified:                    false,
		TaxiParkComment:               record.TaxiParkComment,
	}

	if err := transaction.QueryRow(ctx, `
		INSERT INTO users (
			phone,
			email,
			role,
			registration_type,
			first_name,
			last_name,
			password_hash,
			must_change_password,
			is_phone_confirmed,
			phone_confirmed_at,
			is_email_confirmed,
			is_active
		)
		VALUES ($1, $2, 'driver', 'driver', $3, $4, $5, true, true, now(), false, true)
		RETURNING id`,
		record.Phone,
		nullableString(record.Email),
		nullableString(record.FirstName),
		nullableString(record.LastName),
		record.PasswordHash,
	).Scan(&result.UserID); err != nil {
		if isUniqueViolation(err) {
			return taxiparkapp.CreateDriverResult{}, taxiparkapp.ErrDriverPhoneAlreadyExists
		}
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("insert taxi park driver user: %w", err)
	}

	if err := transaction.QueryRow(ctx, `
		INSERT INTO drivers (
			user_id,
			city_id,
			taxi_park_id,
			status,
			rating,
			ratings_count,
			birth_date,
			license_series,
			license_number,
			license_category,
			license_issued_at,
			license_expires_at,
			driving_experience_from,
			has_no_taxi_work_restrictions,
			federal_law_580_compliant,
			regional_requirements_compliant,
			medical_check_passed,
			pretrip_control_required,
			pretrip_control_passed,
			no_transport_ban,
			verification_status,
			verification_checked_at,
			verification_checked_by,
			is_verified,
			taxi_park_comment
		)
		VALUES ($1, $2, $3, 'offline', 5.00, 0, $4::date, $5, $6, $7, $8::date, $9::date, $10::date, $11, $12, $13, $14, $15, $16, $17,
		        $18::text, CASE WHEN $18::text = 'verified' THEN now() ELSE NULL END, CASE WHEN $18::text = 'verified' THEN $19::uuid ELSE NULL END, $18::text = 'verified', $20)
		RETURNING id`,
		result.UserID,
		cityID,
		taxiParkID,
		nullableDatePtr(record.BirthDate),
		nullableString(record.LicenseSeries),
		nullableString(record.LicenseNumber),
		nullableString(record.LicenseCategory),
		nullableDatePtr(record.LicenseIssuedAt),
		nullableDatePtr(record.LicenseExpiresAt),
		nullableDatePtr(record.DrivingExperienceFrom),
		record.HasNoTaxiWorkRestrictions,
		record.FederalLaw580Compliant,
		record.RegionalRequirementsCompliant,
		record.MedicalCheckPassed,
		record.PretripControlRequired,
		record.PretripControlPassed,
		record.NoTransportBan,
		record.VerificationStatus,
		ownerUserID,
		nullableString(record.TaxiParkComment),
	).Scan(&result.DriverID); err != nil {
		if isUniqueViolation(err) {
			return taxiparkapp.CreateDriverResult{}, taxiparkapp.ErrDriverPhoneAlreadyExists
		}
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("insert taxi park driver profile: %w", err)
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO driver_balances (driver_id, available_balance_cents, pending_balance_cents, currency)
		VALUES ($1, 0, 0, 'RUB')
		ON CONFLICT (driver_id) DO NOTHING`, result.DriverID); err != nil {
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("insert taxi park driver balance: %w", err)
	}
	if record.AttachedCarID != nil {
		if err := repository.assignCarToDriver(ctx, transaction, taxiParkID, *record.AttachedCarID, result.DriverID); err != nil {
			return taxiparkapp.CreateDriverResult{}, err
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("commit create taxi park driver transaction: %w", err)
	}

	return result, nil
}

func (repository *PostgresTaxiParkSettingsRepository) UpdateDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, record taxiparkapp.UpdateDriverRecord) (taxiparkapp.CreateDriverResult, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("begin update taxi park driver transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	result, err := scanTaxiParkDriverResult(transaction.QueryRow(ctx, `
		UPDATE drivers d
		SET birth_date = COALESCE($3::date, d.birth_date),
		    license_series = COALESCE($4::text, d.license_series),
		    license_number = COALESCE($5::text, d.license_number),
		    license_issued_at = COALESCE($6::date, d.license_issued_at),
		    license_expires_at = COALESCE($7::date, d.license_expires_at),
		    driving_experience_from = COALESCE($8::date, d.driving_experience_from),
		    verification_status = COALESCE($9::text, d.verification_status),
		    is_verified = COALESCE(($9::text = 'verified'), d.is_verified),
		    verification_checked_at = CASE WHEN $9::text = 'verified' THEN now() ELSE d.verification_checked_at END,
		    verification_checked_by = CASE WHEN $9::text = 'verified' THEN $1::uuid ELSE d.verification_checked_by END,
		    blocked_reason = CASE WHEN $9::text = 'blocked' THEN COALESCE($10::text, d.blocked_reason) ELSE d.blocked_reason END,
		    taxi_park_comment = COALESCE($10::text, d.taxi_park_comment),
		    license_category = COALESCE($11::text, d.license_category),
		    has_no_taxi_work_restrictions = COALESCE($12::boolean, d.has_no_taxi_work_restrictions),
		    federal_law_580_compliant = COALESCE($13::boolean, d.federal_law_580_compliant),
		    regional_requirements_compliant = COALESCE($14::boolean, d.regional_requirements_compliant),
		    medical_check_passed = COALESCE($15::boolean, d.medical_check_passed),
		    pretrip_control_required = COALESCE($16::boolean, d.pretrip_control_required),
		    pretrip_control_passed = COALESCE($17::boolean, d.pretrip_control_passed),
		    no_transport_ban = COALESCE($18::boolean, d.no_transport_ban)
		FROM taxi_parks tp, users u
		WHERE tp.id = d.taxi_park_id
		  AND u.id = d.user_id
		  AND tp.owner_user_id = $1
		  AND d.id = $2
		  AND tp.deleted_at IS NULL
		  AND d.deleted_at IS NULL
		RETURNING d.id, d.user_id, d.taxi_park_id, u.phone, COALESCE(u.email, ''),
		          COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), d.status, d.verification_status,
		          d.rating::float8, d.ratings_count, d.birth_date, COALESCE(d.license_series, ''),
		          COALESCE(d.license_number, ''), COALESCE(d.license_category, ''), d.license_issued_at, d.license_expires_at,
		          d.driving_experience_from, d.has_no_taxi_work_restrictions, d.federal_law_580_compliant,
		          d.regional_requirements_compliant, d.medical_check_passed, d.pretrip_control_required,
		          d.pretrip_control_passed, d.no_transport_ban, d.verification_checked_at, d.verification_checked_by,
		          d.is_verified, COALESCE(d.taxi_park_comment, '')`,
		ownerUserID,
		driverID,
		nullableDatePtr(record.BirthDate),
		record.LicenseSeries,
		record.LicenseNumber,
		nullableDatePtr(record.LicenseIssuedAt),
		nullableDatePtr(record.LicenseExpiresAt),
		nullableDatePtr(record.DrivingExperienceFrom),
		record.VerificationStatus,
		record.TaxiParkComment,
		record.LicenseCategory,
		record.HasNoTaxiWorkRestrictions,
		record.FederalLaw580Compliant,
		record.RegionalRequirementsCompliant,
		record.MedicalCheckPassed,
		record.PretripControlRequired,
		record.PretripControlPassed,
		record.NoTransportBan,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return taxiparkapp.CreateDriverResult{}, taxiparkapp.ErrTaxiParkResourceNotFound
		}
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("update taxi park driver: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE users u
		SET first_name = COALESCE($3, first_name),
		    last_name = COALESCE($4, last_name)
		FROM drivers d, taxi_parks tp
		WHERE d.user_id = u.id
		  AND tp.id = d.taxi_park_id
		  AND tp.owner_user_id = $1
		  AND d.id = $2
		  AND d.deleted_at IS NULL`,
		ownerUserID,
		driverID,
		record.FirstName,
		record.LastName,
	); err != nil {
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("update taxi park driver user: %w", err)
	}
	if record.AttachedCarID != nil {
		if err := repository.assignCarToDriver(ctx, transaction, result.TaxiParkID, *record.AttachedCarID, result.DriverID); err != nil {
			return taxiparkapp.CreateDriverResult{}, err
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return taxiparkapp.CreateDriverResult{}, fmt.Errorf("commit update taxi park driver transaction: %w", err)
	}
	return result, nil
}

func (repository *PostgresTaxiParkSettingsRepository) ListDriverLocationsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, maxAge time.Duration) ([]taxiparkapp.DriverLocation, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT
			d.id,
			d.user_id,
			trim(concat_ws(' ', COALESCE(u.first_name, ''), COALESCE(u.last_name, ''))) AS driver_name,
			u.phone,
			d.status,
			d.verification_status,
			d.rating::float8,
			CASE WHEN dl.location IS NULL THEN NULL ELSE ST_Y(dl.location::geometry) END AS latitude,
			CASE WHEN dl.location IS NULL THEN NULL ELSE ST_X(dl.location::geometry) END AS longitude,
			dl.heading,
			dl.speed_mps,
			dl.accuracy_meters,
			dl.recorded_at,
			dl.updated_at,
			CASE WHEN dl.updated_at IS NULL THEN true ELSE dl.updated_at < now() - make_interval(secs => $2) END AS is_stale,
			c.id,
			COALESCE(c.brand, ''),
			COALESCE(c.model, ''),
			COALESCE(c.plate_number, ''),
			COALESCE(c.color, ''),
			COALESCE(c.car_class, ''),
			c.verification_status,
			COALESCE(c.is_active, false)
		FROM taxi_parks tp
		JOIN drivers d ON d.taxi_park_id = tp.id
		JOIN users u ON u.id = d.user_id
		LEFT JOIN driver_locations dl ON dl.driver_id = d.id
		LEFT JOIN LATERAL (
			SELECT car.id,
			       car.brand,
			       car.model,
			       car.plate_number,
			       car.color,
			       car.car_class,
			       car.verification_status,
			       car.is_active
			FROM cars car
			LEFT JOIN car_driver_assignments cda ON cda.car_id = car.id
			WHERE car.taxi_park_id = tp.id
			  AND car.deleted_at IS NULL
			  AND (car.driver_id = d.id OR cda.driver_id = d.id)
			ORDER BY (car.driver_id = d.id) DESC, car.is_active DESC, car.updated_at DESC
			LIMIT 1
		) c ON true
		WHERE tp.owner_user_id = $1
		  AND tp.deleted_at IS NULL
		  AND d.deleted_at IS NULL
		  AND u.deleted_at IS NULL
		ORDER BY d.status = 'online' DESC, dl.updated_at DESC NULLS LAST, driver_name ASC`, ownerUserID, int(maxAge.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("select taxi park driver locations: %w", err)
	}
	defer rows.Close()

	locations := make([]taxiparkapp.DriverLocation, 0)
	for rows.Next() {
		location, err := scanTaxiParkDriverLocation(rows)
		if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park driver locations: %w", err)
	}
	return locations, nil
}

func (repository *PostgresTaxiParkSettingsRepository) UpdateDriverPasswordByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, passwordHash string) error {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin taxi park driver password transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	var driverExists bool
	if err := transaction.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM drivers WHERE id = $1 AND deleted_at IS NULL)`, driverID).Scan(&driverExists); err != nil {
		return fmt.Errorf("check taxi park driver password target: %w", err)
	}
	if !driverExists {
		return taxiparkapp.ErrTaxiParkResourceNotFound
	}

	commandTag, err := transaction.Exec(ctx, `
		UPDATE users u
		SET password_hash = $3,
		    must_change_password = false
		FROM drivers d
		JOIN taxi_parks tp ON tp.id = d.taxi_park_id
		WHERE u.id = d.user_id
		  AND tp.owner_user_id = $1
		  AND d.id = $2
		  AND d.deleted_at IS NULL
		  AND tp.deleted_at IS NULL`, ownerUserID, driverID, passwordHash)
	if err != nil {
		return fmt.Errorf("update taxi park driver password: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return taxiparkapp.ErrTaxiParkResourceForbidden
	}

	payload, err := json.Marshal(map[string]any{
		"driver_id": driverID,
	})
	if err != nil {
		return fmt.Errorf("marshal taxi park driver password audit payload: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO finance_audit_events (actor_user_id, event_type, payload)
		VALUES ($1, 'taxi_park.driver.password_updated', $2)`, ownerUserID, payload); err != nil {
		return fmt.Errorf("insert taxi park driver password audit event: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit taxi park driver password transaction: %w", err)
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) BlockDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, reason string) error {
	commandTag, err := repository.pool.Exec(ctx, `
		UPDATE drivers d
		SET verification_status = 'blocked',
		    status = 'blocked',
		    blocked_reason = $3,
		    taxi_park_comment = $3
		FROM taxi_parks tp
		WHERE tp.id = d.taxi_park_id
		  AND tp.owner_user_id = $1
		  AND d.id = $2
		  AND d.deleted_at IS NULL`, ownerUserID, driverID, nullableString(reason))
	if err != nil {
		return fmt.Errorf("block taxi park driver: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return taxiparkapp.ErrTaxiParkNotFound
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) ArchiveDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin archive taxi park driver transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)
	var userID uuid.UUID
	if err := transaction.QueryRow(ctx, `
		UPDATE drivers d
		SET verification_status = 'archived',
		    status = 'blocked',
		    deleted_at = now()
		FROM taxi_parks tp
		WHERE tp.id = d.taxi_park_id
		  AND tp.owner_user_id = $1
		  AND d.id = $2
		  AND d.deleted_at IS NULL
		RETURNING d.user_id`, ownerUserID, driverID).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return taxiparkapp.ErrTaxiParkResourceNotFound
		}
		return fmt.Errorf("archive taxi park driver: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE cars c
		SET driver_id = NULL,
		    updated_at = now()
		FROM taxi_parks tp
		WHERE tp.id = c.taxi_park_id
		  AND tp.owner_user_id = $1
		  AND c.driver_id = $2
		  AND c.deleted_at IS NULL`, ownerUserID, driverID); err != nil {
		return fmt.Errorf("detach primary cars before archive driver: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		DELETE FROM car_driver_assignments cda
		USING cars c, taxi_parks tp
		WHERE c.id = cda.car_id
		  AND tp.id = c.taxi_park_id
		  AND tp.owner_user_id = $1
		  AND cda.driver_id = $2`, ownerUserID, driverID); err != nil {
		return fmt.Errorf("delete car assignments before archive driver: %w", err)
	}
	if _, err := transaction.Exec(ctx, `UPDATE users SET is_active = false, deleted_at = now() WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("archive taxi park driver user: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit archive taxi park driver transaction: %w", err)
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) ListCarsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) ([]domain.Car, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT c.id, c.taxi_park_id, c.driver_id, c.brand, c.model, COALESCE(c.year, 0),
		       c.plate_number, COALESCE(c.vin, ''), COALESCE(c.sts, ''), COALESCE(c.pts, ''),
		       c.color, COALESCE(c.car_class, ''), c.verification_status, COALESCE(c.owner_details, ''),
		       c.osago_expires_at, c.diagnostic_card_expires_at, COALESCE(c.taxi_permit_number, ''),
		       COALESCE(c.regional_registry_number, ''), COALESCE(c.permit_region, ''),
		       c.permit_issued_at, c.permit_expires_at, c.taxi_permit_verified, c.regional_registry_verified,
		       c.regional_requirements_compliant, c.has_taxi_color_scheme, c.has_orange_roof_lamp,
		       c.has_passenger_info, c.osago_verified, c.diagnostic_card_verified, c.technical_state_verified,
		       c.localization_compliant, c.legal_use_basis_verified, c.verification_checked_at,
		       c.verification_checked_by, c.is_active, c.created_at, c.updated_at
		FROM cars c
		JOIN taxi_parks tp ON tp.id = c.taxi_park_id
		WHERE tp.owner_user_id = $1 AND c.deleted_at IS NULL AND tp.deleted_at IS NULL
		ORDER BY c.created_at DESC`, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list taxi park cars: %w", err)
	}
	defer rows.Close()
	return repository.scanCars(ctx, rows)
}

func (repository *PostgresTaxiParkSettingsRepository) ListCarsByDriverAndOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) ([]domain.Car, error) {
	var driverExists bool
	if err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM drivers d
			JOIN taxi_parks tp ON tp.id = d.taxi_park_id
			WHERE tp.owner_user_id = $1
			  AND d.id = $2
			  AND tp.deleted_at IS NULL
			  AND d.deleted_at IS NULL
		)`, ownerUserID, driverID).Scan(&driverExists); err != nil {
		return nil, fmt.Errorf("check taxi park driver before list cars: %w", err)
	}
	if !driverExists {
		return nil, taxiparkapp.ErrTaxiParkResourceNotFound
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT DISTINCT c.id, c.taxi_park_id, c.driver_id, c.brand, c.model, COALESCE(c.year, 0),
		       c.plate_number, COALESCE(c.vin, ''), COALESCE(c.sts, ''), COALESCE(c.pts, ''),
		       c.color, COALESCE(c.car_class, ''), c.verification_status, COALESCE(c.owner_details, ''),
		       c.osago_expires_at, c.diagnostic_card_expires_at, COALESCE(c.taxi_permit_number, ''),
		       COALESCE(c.regional_registry_number, ''), COALESCE(c.permit_region, ''),
		       c.permit_issued_at, c.permit_expires_at, c.taxi_permit_verified, c.regional_registry_verified,
		       c.regional_requirements_compliant, c.has_taxi_color_scheme, c.has_orange_roof_lamp,
		       c.has_passenger_info, c.osago_verified, c.diagnostic_card_verified, c.technical_state_verified,
		       c.localization_compliant, c.legal_use_basis_verified, c.verification_checked_at,
		       c.verification_checked_by, c.is_active, c.created_at, c.updated_at
		FROM cars c
		JOIN taxi_parks tp ON tp.id = c.taxi_park_id
		LEFT JOIN car_driver_assignments cda ON cda.car_id = c.id
		WHERE tp.owner_user_id = $1
		  AND c.deleted_at IS NULL
		  AND tp.deleted_at IS NULL
		  AND (c.driver_id = $2 OR cda.driver_id = $2)
		ORDER BY c.created_at DESC`, ownerUserID, driverID)
	if err != nil {
		return nil, fmt.Errorf("list taxi park driver cars: %w", err)
	}
	defer rows.Close()
	return repository.scanCars(ctx, rows)
}

func (repository *PostgresTaxiParkSettingsRepository) CreateCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record taxiparkapp.CarRecord) (domain.Car, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Car{}, fmt.Errorf("begin create taxi park car transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)
	taxiParkID, err := taxiParkIDByOwner(ctx, transaction, ownerUserID)
	if err != nil {
		return domain.Car{}, err
	}
	car, err := scanTaxiParkCar(transaction.QueryRow(ctx, `
		INSERT INTO cars (
			taxi_park_id, driver_id, brand, model, year, plate_number, vin, sts, pts, color,
			car_class, verification_status, owner_details, osago_expires_at, diagnostic_card_expires_at,
			taxi_permit_number, regional_registry_number, permit_region, permit_issued_at, permit_expires_at,
			taxi_permit_verified, regional_registry_verified, regional_requirements_compliant, has_taxi_color_scheme,
			has_orange_roof_lamp, has_passenger_info, osago_verified, diagnostic_card_verified, technical_state_verified,
			localization_compliant, legal_use_basis_verified, verification_checked_at, verification_checked_by, is_active
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, 0), $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		        $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, CASE WHEN $12 = 'verified' THEN now() ELSE NULL END,
		        CASE WHEN $12 = 'verified' THEN $32::uuid ELSE NULL END, $33)
		RETURNING id, taxi_park_id, driver_id, brand, model, COALESCE(year, 0), plate_number, COALESCE(vin, ''),
		          COALESCE(sts, ''), COALESCE(pts, ''), color, COALESCE(car_class, ''), verification_status,
		          COALESCE(owner_details, ''), osago_expires_at, diagnostic_card_expires_at,
		          COALESCE(taxi_permit_number, ''), COALESCE(regional_registry_number, ''), COALESCE(permit_region, ''),
		          permit_issued_at, permit_expires_at, taxi_permit_verified, regional_registry_verified,
		          regional_requirements_compliant, has_taxi_color_scheme, has_orange_roof_lamp, has_passenger_info,
		          osago_verified, diagnostic_card_verified, technical_state_verified, localization_compliant,
		          legal_use_basis_verified, verification_checked_at, verification_checked_by, is_active, created_at, updated_at`,
		taxiParkID, nullableUUIDPtr(record.PrimaryDriverID), record.Brand, record.Model, record.Year, record.PlateNumber,
		nullableString(record.VIN), nullableString(record.STS), nullableString(record.PTS), record.Color,
		nullableString(record.CarClass), record.VerificationStatus, nullableString(record.OwnerDetails),
		nullableDatePtr(record.OSAGOExpiresAt), nullableDatePtr(record.DiagnosticCardExpiresAt), nullableString(record.TaxiPermitNumber),
		nullableString(record.RegionalRegistryNumber), nullableString(record.PermitRegion), nullableDatePtr(record.PermitIssuedAt),
		nullableDatePtr(record.PermitExpiresAt), record.TaxiPermitVerified, record.RegionalRegistryVerified,
		record.RegionalRequirementsCompliant, record.HasTaxiColorScheme, record.HasOrangeRoofLamp,
		record.HasPassengerInfo, record.OSAGOVerified, record.DiagnosticCardVerified,
		record.TechnicalStateVerified, record.LocalizationCompliant, record.LegalUseBasisVerified,
		ownerUserID, record.IsActive,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Car{}, taxiparkapp.ErrCarAlreadyExists
		}
		return domain.Car{}, fmt.Errorf("insert taxi park car: %w", err)
	}
	if err := repository.replaceCarAssignments(ctx, transaction, taxiParkID, car.ID, record.AttachedDriverIDs); err != nil {
		return domain.Car{}, err
	}
	car.AttachedDriverIDs = record.AttachedDriverIDs
	if err := transaction.Commit(ctx); err != nil {
		return domain.Car{}, fmt.Errorf("commit create taxi park car transaction: %w", err)
	}
	return car, nil
}

func (repository *PostgresTaxiParkSettingsRepository) UpdateCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID, record taxiparkapp.CarPatchRecord) (domain.Car, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Car{}, fmt.Errorf("begin update taxi park car transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)
	taxiParkID, err := taxiParkIDByOwner(ctx, transaction, ownerUserID)
	if err != nil {
		return domain.Car{}, err
	}
	car, err := scanTaxiParkCar(transaction.QueryRow(ctx, `
		UPDATE cars
		SET driver_id = COALESCE($3, driver_id),
		    brand = COALESCE($4, brand),
		    model = COALESCE($5, model),
		    year = COALESCE($6, year),
		    plate_number = COALESCE($7, plate_number),
		    vin = COALESCE($8, vin),
		    sts = COALESCE($9, sts),
		    pts = COALESCE($10, pts),
		    color = COALESCE($11, color),
		    car_class = COALESCE($12, car_class),
		    verification_status = COALESCE($13, verification_status),
		    owner_details = COALESCE($14, owner_details),
		    osago_expires_at = COALESCE($15, osago_expires_at),
		    diagnostic_card_expires_at = COALESCE($16, diagnostic_card_expires_at),
		    taxi_permit_number = COALESCE($17, taxi_permit_number),
		    regional_registry_number = COALESCE($18, regional_registry_number),
		    permit_region = COALESCE($19, permit_region),
		    permit_issued_at = COALESCE($20, permit_issued_at),
		    permit_expires_at = COALESCE($21, permit_expires_at),
		    taxi_permit_verified = COALESCE($22, taxi_permit_verified),
		    regional_registry_verified = COALESCE($23, regional_registry_verified),
		    regional_requirements_compliant = COALESCE($24, regional_requirements_compliant),
		    has_taxi_color_scheme = COALESCE($25, has_taxi_color_scheme),
		    has_orange_roof_lamp = COALESCE($26, has_orange_roof_lamp),
		    has_passenger_info = COALESCE($27, has_passenger_info),
		    osago_verified = COALESCE($28, osago_verified),
		    diagnostic_card_verified = COALESCE($29, diagnostic_card_verified),
		    technical_state_verified = COALESCE($30, technical_state_verified),
		    localization_compliant = COALESCE($31, localization_compliant),
		    legal_use_basis_verified = COALESCE($32, legal_use_basis_verified),
		    verification_checked_at = CASE WHEN $13 = 'verified' THEN now() ELSE verification_checked_at END,
		    verification_checked_by = CASE WHEN $13 = 'verified' THEN $33::uuid ELSE verification_checked_by END,
		    is_active = COALESCE($34, is_active)
		WHERE taxi_park_id = $1 AND id = $2 AND deleted_at IS NULL
		RETURNING id, taxi_park_id, driver_id, brand, model, COALESCE(year, 0), plate_number, COALESCE(vin, ''),
		          COALESCE(sts, ''), COALESCE(pts, ''), color, COALESCE(car_class, ''), verification_status,
		          COALESCE(owner_details, ''), osago_expires_at, diagnostic_card_expires_at,
		          COALESCE(taxi_permit_number, ''), COALESCE(regional_registry_number, ''), COALESCE(permit_region, ''),
		          permit_issued_at, permit_expires_at, taxi_permit_verified, regional_registry_verified,
		          regional_requirements_compliant, has_taxi_color_scheme, has_orange_roof_lamp, has_passenger_info,
		          osago_verified, diagnostic_card_verified, technical_state_verified, localization_compliant,
		          legal_use_basis_verified, verification_checked_at, verification_checked_by, is_active, created_at, updated_at`,
		taxiParkID, carID, nullableUUIDPtr(record.PrimaryDriverID), record.Brand, record.Model, nullableIntPtr(record.Year), record.PlateNumber,
		record.VIN, record.STS, record.PTS, record.Color, record.CarClass, record.VerificationStatus,
		record.OwnerDetails, nullableDatePtr(record.OSAGOExpiresAt), nullableDatePtr(record.DiagnosticCardExpiresAt), record.TaxiPermitNumber,
		record.RegionalRegistryNumber, record.PermitRegion, nullableDatePtr(record.PermitIssuedAt), nullableDatePtr(record.PermitExpiresAt),
		record.TaxiPermitVerified, record.RegionalRegistryVerified, record.RegionalRequirementsCompliant,
		record.HasTaxiColorScheme, record.HasOrangeRoofLamp, record.HasPassengerInfo, record.OSAGOVerified,
		record.DiagnosticCardVerified, record.TechnicalStateVerified, record.LocalizationCompliant,
		record.LegalUseBasisVerified, ownerUserID, record.IsActive,
	))
	if err != nil {
		return domain.Car{}, fmt.Errorf("update taxi park car: %w", err)
	}
	if record.AttachedDriverIDs != nil {
		if err := repository.replaceCarAssignments(ctx, transaction, taxiParkID, car.ID, record.AttachedDriverIDs); err != nil {
			return domain.Car{}, err
		}
		car.AttachedDriverIDs = record.AttachedDriverIDs
	}
	if err := transaction.Commit(ctx); err != nil {
		return domain.Car{}, fmt.Errorf("commit update taxi park car transaction: %w", err)
	}
	return car, nil
}

func (repository *PostgresTaxiParkSettingsRepository) UnblockDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error {
	commandTag, err := repository.pool.Exec(ctx, `
		UPDATE drivers d
		SET status = 'offline',
		    verification_status = CASE WHEN d.is_verified THEN 'verified' ELSE 'pending_verification' END,
		    blocked_reason = NULL
		FROM taxi_parks tp
		WHERE tp.id = d.taxi_park_id
		  AND tp.owner_user_id = $1
		  AND d.id = $2
		  AND d.verification_status <> 'archived'
		  AND d.deleted_at IS NULL
		  AND tp.deleted_at IS NULL`, ownerUserID, driverID)
	if err != nil {
		return fmt.Errorf("unblock taxi park driver: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return taxiparkapp.ErrTaxiParkResourceNotFound
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) ArchiveCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) error {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin archive taxi park car transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	taxiParkID, err := taxiParkIDByOwner(ctx, transaction, ownerUserID)
	if err != nil {
		return err
	}

	commandTag, err := transaction.Exec(ctx, `
		UPDATE cars
		SET verification_status = 'archived',
		    is_active = false,
		    deleted_at = now()
		WHERE taxi_park_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`, taxiParkID, carID)
	if err != nil {
		return fmt.Errorf("archive taxi park car: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return taxiparkapp.ErrTaxiParkResourceNotFound
	}

	if _, err := transaction.Exec(ctx, `DELETE FROM car_driver_assignments WHERE taxi_park_id = $1 AND car_id = $2`, taxiParkID, carID); err != nil {
		return fmt.Errorf("delete archived car assignments: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit archive taxi park car transaction: %w", err)
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) AttachCarToDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error {
	return repository.AssignCarToDriverByOwnerUserID(ctx, ownerUserID, driverID, carID)
}

func (repository *PostgresTaxiParkSettingsRepository) AssignCarToDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin assign taxi park car transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	taxiParkID, err := taxiParkIDByOwner(ctx, transaction, ownerUserID)
	if err != nil {
		return err
	}
	if err := repository.ensureDriverAndCarBelongToPark(ctx, transaction, taxiParkID, driverID, carID); err != nil {
		return err
	}
	if err := repository.assignCarToDriver(ctx, transaction, taxiParkID, carID, driverID); err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE cars
		SET driver_id = $3,
		    updated_at = now()
		WHERE taxi_park_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`, taxiParkID, carID, driverID); err != nil {
		return fmt.Errorf("assign taxi park car primary driver: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit assign taxi park car transaction: %w", err)
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) DetachCarFromDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin detach taxi park car transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	taxiParkID, err := taxiParkIDByOwner(ctx, transaction, ownerUserID)
	if err != nil {
		return err
	}
	if err := repository.ensureDriverAndCarBelongToPark(ctx, transaction, taxiParkID, driverID, carID); err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, `DELETE FROM car_driver_assignments WHERE taxi_park_id = $1 AND car_id = $2 AND driver_id = $3`, taxiParkID, carID, driverID); err != nil {
		return fmt.Errorf("detach taxi park car from driver: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE cars
		SET driver_id = (
			SELECT driver_id
			FROM car_driver_assignments
			WHERE taxi_park_id = $1 AND car_id = $2
			ORDER BY created_at
			LIMIT 1
		)
		WHERE taxi_park_id = $1 AND id = $2 AND driver_id = $3`, taxiParkID, carID, driverID); err != nil {
		return fmt.Errorf("reset taxi park car primary driver: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit detach taxi park car transaction: %w", err)
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) ListDriverDocumentsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) ([]domain.TaxiParkDocument, error) {
	var exists bool
	if err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM drivers d
			JOIN taxi_parks tp ON tp.id = d.taxi_park_id
			WHERE tp.owner_user_id = $1
			  AND d.id = $2
			  AND d.deleted_at IS NULL
			  AND tp.deleted_at IS NULL
		)`, ownerUserID, driverID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check taxi park driver documents owner: %w", err)
	}
	if !exists {
		return nil, taxiparkapp.ErrTaxiParkResourceNotFound
	}
	return []domain.TaxiParkDocument{}, nil
}

func (repository *PostgresTaxiParkSettingsRepository) ListCarDocumentsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) ([]domain.TaxiParkDocument, error) {
	var exists bool
	if err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM cars c
			JOIN taxi_parks tp ON tp.id = c.taxi_park_id
			WHERE tp.owner_user_id = $1
			  AND c.id = $2
			  AND c.deleted_at IS NULL
			  AND tp.deleted_at IS NULL
		)`, ownerUserID, carID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check taxi park car documents owner: %w", err)
	}
	if !exists {
		return nil, taxiparkapp.ErrTaxiParkResourceNotFound
	}
	return []domain.TaxiParkDocument{}, nil
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

func scanTaxiParkDriverResult(row pgx.Row) (taxiparkapp.CreateDriverResult, error) {
	var result taxiparkapp.CreateDriverResult
	var birthDate pgtype.Date
	var licenseIssuedAt pgtype.Date
	var licenseExpiresAt pgtype.Date
	var drivingExperienceFrom pgtype.Date
	var verificationCheckedAt pgtype.Timestamptz
	var verificationCheckedBy pgtype.UUID
	if err := row.Scan(
		&result.DriverID,
		&result.UserID,
		&result.TaxiParkID,
		&result.Phone,
		&result.Email,
		&result.FirstName,
		&result.LastName,
		&result.Status,
		&result.VerificationStatus,
		&result.Rating,
		&result.RatingsCount,
		&birthDate,
		&result.LicenseSeries,
		&result.LicenseNumber,
		&result.LicenseCategory,
		&licenseIssuedAt,
		&licenseExpiresAt,
		&drivingExperienceFrom,
		&result.HasNoTaxiWorkRestrictions,
		&result.FederalLaw580Compliant,
		&result.RegionalRequirementsCompliant,
		&result.MedicalCheckPassed,
		&result.PretripControlRequired,
		&result.PretripControlPassed,
		&result.NoTransportBan,
		&verificationCheckedAt,
		&verificationCheckedBy,
		&result.IsVerified,
		&result.TaxiParkComment,
	); err != nil {
		return taxiparkapp.CreateDriverResult{}, err
	}
	result.BirthDate = datePtr(birthDate)
	result.LicenseIssuedAt = datePtr(licenseIssuedAt)
	result.LicenseExpiresAt = datePtr(licenseExpiresAt)
	result.DrivingExperienceFrom = datePtr(drivingExperienceFrom)
	if verificationCheckedAt.Valid {
		result.VerificationCheckedAt = &verificationCheckedAt.Time
	}
	if verificationCheckedBy.Valid {
		value, err := uuid.FromBytes(verificationCheckedBy.Bytes[:])
		if err != nil {
			return taxiparkapp.CreateDriverResult{}, err
		}
		result.VerificationCheckedBy = &value
	}
	return result, nil
}

func scanTaxiParkDriverLocation(row pgx.Row) (taxiparkapp.DriverLocation, error) {
	var location taxiparkapp.DriverLocation
	var latitude pgtype.Float8
	var longitude pgtype.Float8
	var heading pgtype.Int2
	var speedMPS pgtype.Float8
	var accuracyMeters pgtype.Float8
	var recordedAt pgtype.Timestamptz
	var updatedAt pgtype.Timestamptz
	var carID pgtype.UUID
	var car taxiparkapp.DriverLocationCar
	var carVerificationStatus pgtype.Text
	var carIsActive bool

	if err := row.Scan(
		&location.DriverID,
		&location.UserID,
		&location.Name,
		&location.Phone,
		&location.Status,
		&location.VerificationStatus,
		&location.Rating,
		&latitude,
		&longitude,
		&heading,
		&speedMPS,
		&accuracyMeters,
		&recordedAt,
		&updatedAt,
		&location.IsStale,
		&carID,
		&car.Brand,
		&car.Model,
		&car.PlateNumber,
		&car.Color,
		&car.CarClass,
		&carVerificationStatus,
		&carIsActive,
	); err != nil {
		return taxiparkapp.DriverLocation{}, fmt.Errorf("scan taxi park driver location: %w", err)
	}

	if latitude.Valid && longitude.Valid {
		coordinates, err := domain.NewCoordinates(latitude.Float64, longitude.Float64)
		if err != nil {
			return taxiparkapp.DriverLocation{}, fmt.Errorf("build taxi park driver location coordinates: %w", err)
		}
		location.Location = &coordinates
	}
	if heading.Valid {
		value := heading.Int16
		location.Heading = &value
	}
	if speedMPS.Valid {
		value := speedMPS.Float64
		location.SpeedMPS = &value
	}
	if accuracyMeters.Valid {
		value := accuracyMeters.Float64
		location.AccuracyMeters = &value
	}
	if recordedAt.Valid {
		location.RecordedAt = &recordedAt.Time
	}
	if updatedAt.Valid {
		location.UpdatedAt = &updatedAt.Time
	}
	if carID.Valid {
		car.ID = uuid.UUID(carID.Bytes)
		car.VerificationStatus = domain.VerificationLifecycleStatus(carVerificationStatus.String)
		car.IsActive = carIsActive
		location.Car = &car
	}
	return location, nil
}

func scanTaxiParkCar(row pgx.Row) (domain.Car, error) {
	var car domain.Car
	var primaryDriverID pgtype.UUID
	var osagoExpiresAt pgtype.Date
	var diagnosticExpiresAt pgtype.Date
	var permitIssuedAt pgtype.Date
	var permitExpiresAt pgtype.Date
	var verificationCheckedAt pgtype.Timestamptz
	var verificationCheckedBy pgtype.UUID
	if err := row.Scan(
		&car.ID,
		&car.TaxiParkID,
		&primaryDriverID,
		&car.Brand,
		&car.Model,
		&car.Year,
		&car.PlateNumber,
		&car.VIN,
		&car.STS,
		&car.PTS,
		&car.Color,
		&car.CarClass,
		&car.VerificationStatus,
		&car.OwnerDetails,
		&osagoExpiresAt,
		&diagnosticExpiresAt,
		&car.TaxiPermitNumber,
		&car.RegionalRegistryNumber,
		&car.PermitRegion,
		&permitIssuedAt,
		&permitExpiresAt,
		&car.TaxiPermitVerified,
		&car.RegionalRegistryVerified,
		&car.RegionalRequirementsCompliant,
		&car.HasTaxiColorScheme,
		&car.HasOrangeRoofLamp,
		&car.HasPassengerInfo,
		&car.OSAGOVerified,
		&car.DiagnosticCardVerified,
		&car.TechnicalStateVerified,
		&car.LocalizationCompliant,
		&car.LegalUseBasisVerified,
		&verificationCheckedAt,
		&verificationCheckedBy,
		&car.IsActive,
		&car.CreatedAt,
		&car.UpdatedAt,
	); err != nil {
		return domain.Car{}, err
	}
	if primaryDriverID.Valid {
		value, err := uuid.FromBytes(primaryDriverID.Bytes[:])
		if err != nil {
			return domain.Car{}, err
		}
		car.PrimaryDriverID = &value
	}
	car.OSAGOExpiresAt = datePtr(osagoExpiresAt)
	car.DiagnosticCardExpiresAt = datePtr(diagnosticExpiresAt)
	car.PermitIssuedAt = datePtr(permitIssuedAt)
	car.PermitExpiresAt = datePtr(permitExpiresAt)
	if verificationCheckedAt.Valid {
		car.VerificationCheckedAt = &verificationCheckedAt.Time
	}
	if verificationCheckedBy.Valid {
		value, err := uuid.FromBytes(verificationCheckedBy.Bytes[:])
		if err != nil {
			return domain.Car{}, err
		}
		car.VerificationCheckedBy = &value
	}
	return car, nil
}

func (repository *PostgresTaxiParkSettingsRepository) scanCars(ctx context.Context, rows pgx.Rows) ([]domain.Car, error) {
	cars := make([]domain.Car, 0)
	for rows.Next() {
		car, err := scanTaxiParkCar(rows)
		if err != nil {
			return nil, err
		}
		attached, err := repository.carAssignments(ctx, car.ID)
		if err != nil {
			return nil, err
		}
		car.AttachedDriverIDs = attached
		cars = append(cars, car)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park cars: %w", err)
	}
	return cars, nil
}

func datePtr(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullableUUIDPtr(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableIntPtr(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableDatePtr(value *time.Time) any {
	if value == nil {
		return nil
	}
	return pgtype.Date{Time: *value, Valid: true}
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func (repository *PostgresTaxiParkSettingsRepository) resolveOrderTariff(ctx context.Context, transaction pgx.Tx, taxiParkID uuid.UUID, tariffID uuid.UUID) (*uuid.UUID, *uuid.UUID, int64, int64, int64, error) {
	var taxiParkTariffID uuid.UUID
	var basePriceCents int64
	var pricePerKMCents int64
	var minimumPriceCents int64
	err := transaction.QueryRow(ctx, `
		SELECT id, base_price_cents, price_per_km_cents, minimum_price_cents
		FROM taxi_park_tariffs
		WHERE id = $1
		  AND taxi_park_id = $2
		  AND is_active = true`, tariffID, taxiParkID).Scan(&taxiParkTariffID, &basePriceCents, &pricePerKMCents, &minimumPriceCents)
	if err == nil {
		return nil, &taxiParkTariffID, basePriceCents, pricePerKMCents, minimumPriceCents, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, 0, 0, 0, fmt.Errorf("select taxi park order tariff: %w", err)
	}

	var orderTariffID uuid.UUID
	err = transaction.QueryRow(ctx, `
		SELECT id, (base_price * 100)::bigint, (price_per_km * 100)::bigint, (minimum_price * 100)::bigint
		FROM tariffs
		WHERE id = $1
		  AND is_active = true
		  AND deleted_at IS NULL`, tariffID).Scan(&orderTariffID, &basePriceCents, &pricePerKMCents, &minimumPriceCents)
	if err == nil {
		return &orderTariffID, nil, basePriceCents, pricePerKMCents, minimumPriceCents, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, 0, 0, 0, fmt.Errorf("%w: tariff_id %s", taxiparkapp.ErrOrderTariffNotFound, tariffID)
	}
	return nil, nil, 0, 0, 0, fmt.Errorf("select global order tariff: %w", err)
}

func splitPassengerName(name string) (string, string) {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func taxiParkIDByOwner(ctx context.Context, transaction pgx.Tx, ownerUserID uuid.UUID) (uuid.UUID, error) {
	var taxiParkID uuid.UUID
	if err := transaction.QueryRow(ctx, `
		SELECT id
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL`, ownerUserID).Scan(&taxiParkID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, taxiparkapp.ErrTaxiParkNotFound
		}
		return uuid.Nil, fmt.Errorf("select taxi park by owner: %w", err)
	}
	return taxiParkID, nil
}

func (repository *PostgresTaxiParkSettingsRepository) assignCarToDriver(ctx context.Context, transaction pgx.Tx, taxiParkID uuid.UUID, carID uuid.UUID, driverID uuid.UUID) error {
	if err := repository.ensureDriverAndCarBelongToPark(ctx, transaction, taxiParkID, driverID, carID); err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO car_driver_assignments (car_id, driver_id, taxi_park_id)
		VALUES ($2, $3, $1)
		ON CONFLICT (car_id, driver_id) DO NOTHING`, taxiParkID, carID, driverID); err != nil {
		return fmt.Errorf("assign taxi park car to driver: %w", err)
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) ensureDriverAndCarBelongToPark(ctx context.Context, transaction pgx.Tx, taxiParkID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error {
	var driverExists bool
	var carExists bool
	if err := transaction.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM drivers WHERE taxi_park_id = $1 AND id = $2 AND deleted_at IS NULL),
			EXISTS (SELECT 1 FROM cars WHERE taxi_park_id = $1 AND id = $3 AND deleted_at IS NULL)`, taxiParkID, driverID, carID).Scan(&driverExists, &carExists); err != nil {
		return fmt.Errorf("check taxi park driver car ownership: %w", err)
	}
	if !driverExists || !carExists {
		return taxiparkapp.ErrTaxiParkResourceNotFound
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) replaceCarAssignments(ctx context.Context, transaction pgx.Tx, taxiParkID uuid.UUID, carID uuid.UUID, driverIDs []uuid.UUID) error {
	if _, err := transaction.Exec(ctx, `DELETE FROM car_driver_assignments WHERE taxi_park_id = $1 AND car_id = $2`, taxiParkID, carID); err != nil {
		return fmt.Errorf("delete car assignments: %w", err)
	}
	for _, driverID := range driverIDs {
		if err := repository.assignCarToDriver(ctx, transaction, taxiParkID, carID, driverID); err != nil {
			return err
		}
	}
	return nil
}

func (repository *PostgresTaxiParkSettingsRepository) carAssignments(ctx context.Context, carID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := repository.pool.Query(ctx, `SELECT driver_id FROM car_driver_assignments WHERE car_id = $1`, carID)
	if err != nil {
		return nil, fmt.Errorf("select car assignments: %w", err)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan car assignment: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
