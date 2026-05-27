package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/finance"
)

type PostgresFinanceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresFinanceRepository(pool *pgxpool.Pool) *PostgresFinanceRepository {
	return &PostgresFinanceRepository{pool: pool}
}

func (repository *PostgresFinanceRepository) GetOrderSnapshot(ctx context.Context, orderID uuid.UUID) (finance.OrderSnapshot, error) {
	const query = `
		SELECT
			o.id,
			o.driver_id,
			d.taxi_park_id,
			o.city_id,
			o.tariff_id,
			o.status,
			CASE WHEN o.final_price IS NULL THEN NULL ELSE (o.final_price * 100)::bigint END,
			(d.commission_percent * 100)::integer,
			(tp.commission_percent * 100)::integer,
			(t.commission_percent * 100)::integer,
			(c.commission_percent * 100)::integer
		FROM orders o
		LEFT JOIN drivers d ON d.id = o.driver_id
		LEFT JOIN taxi_parks tp ON tp.id = d.taxi_park_id
		LEFT JOIN tariffs t ON t.id = o.tariff_id
		LEFT JOIN cities c ON c.id = o.city_id
		WHERE o.id = $1 AND o.deleted_at IS NULL`

	var snapshot finance.OrderSnapshot
	var driverID pgtype.UUID
	var taxiParkID pgtype.UUID
	var tariffID pgtype.UUID
	var finalPriceCents pgtype.Int8
	var driverBPS pgtype.Int4
	var taxiParkBPS pgtype.Int4
	var tariffBPS pgtype.Int4
	var cityBPS pgtype.Int4

	if err := repository.pool.QueryRow(ctx, query, orderID).Scan(
		&snapshot.OrderID,
		&driverID,
		&taxiParkID,
		&snapshot.CityID,
		&tariffID,
		&snapshot.Status,
		&finalPriceCents,
		&driverBPS,
		&taxiParkBPS,
		&tariffBPS,
		&cityBPS,
	); err != nil {
		return finance.OrderSnapshot{}, fmt.Errorf("select finance order snapshot: %w", err)
	}

	if driverID.Valid {
		value := uuid.UUID(driverID.Bytes)
		snapshot.DriverID = &value
	}
	if taxiParkID.Valid {
		value := uuid.UUID(taxiParkID.Bytes)
		snapshot.TaxiParkID = &value
	}
	if tariffID.Valid {
		value := uuid.UUID(tariffID.Bytes)
		snapshot.TariffID = &value
	}
	if finalPriceCents.Valid {
		money := domain.Money{Amount: finalPriceCents.Int64, Currency: "RUB"}
		snapshot.FinalPrice = &money
	}
	snapshot.DriverCommissionBPS = optionalInt32(driverBPS)
	snapshot.TaxiParkCommissionBPS = optionalInt32(taxiParkBPS)
	snapshot.TariffCommissionBPS = optionalInt32(tariffBPS)
	snapshot.CityCommissionBPS = optionalInt32(cityBPS)
	snapshot.PlatformDefaultBPS = domain.DefaultPlatformCommissionBasisPoints

	return snapshot, nil
}

func (repository *PostgresFinanceRepository) CreateOrderSettlement(ctx context.Context, settlement domain.OrderSettlement) (domain.OrderSettlement, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("begin finance transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	var existingID uuid.UUID
	existingErr := tx.QueryRow(ctx, `
		SELECT id
		FROM financial_transactions
		WHERE order_id = $1 AND transaction_type = 'driver_income'
		LIMIT 1
		FOR SHARE`, settlement.OrderID).Scan(&existingID)
	if existingErr == nil {
		return domain.OrderSettlement{}, finance.ErrFinancialSettlementDuplicate
	}
	if existingErr != nil && !errors.Is(existingErr, pgx.ErrNoRows) {
		return domain.OrderSettlement{}, fmt.Errorf("check duplicate settlement: %w", existingErr)
	}

	commissionTransactionID, err := insertFinancialTransaction(ctx, tx, settlement, domain.TransactionTypeCommission)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.OrderSettlement{}, finance.ErrFinancialSettlementDuplicate
		}
		return domain.OrderSettlement{}, err
	}
	incomeTransactionID, err := insertFinancialTransaction(ctx, tx, settlement, domain.TransactionTypeDriverIncome)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.OrderSettlement{}, finance.ErrFinancialSettlementDuplicate
		}
		return domain.OrderSettlement{}, err
	}

	if settlement.TaxiParkID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE taxi_parks
			SET balance_cents = balance_cents + $2
			WHERE id = $1`, *settlement.TaxiParkID, settlement.NetAmount.Amount); err != nil {
			return domain.OrderSettlement{}, fmt.Errorf("update taxi park balance: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO driver_balances (driver_id, available_balance_cents, pending_balance_cents, currency)
			VALUES ($1, $2, 0, $3)
			ON CONFLICT (driver_id) DO UPDATE
			SET available_balance_cents = driver_balances.available_balance_cents + EXCLUDED.available_balance_cents,
			    version = driver_balances.version + 1`,
			settlement.DriverID,
			settlement.NetAmount.Amount,
			settlement.NetAmount.Currency,
		); err != nil {
			return domain.OrderSettlement{}, fmt.Errorf("upsert driver balance: %w", err)
		}
	}

	if err := insertFinanceAudit(ctx, tx, commissionTransactionID, settlement, "commission_created"); err != nil {
		return domain.OrderSettlement{}, err
	}
	if err := insertFinanceAudit(ctx, tx, incomeTransactionID, settlement, "income_created"); err != nil {
		return domain.OrderSettlement{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("commit finance transaction: %w", err)
	}

	settlement.CommissionTransactionID = commissionTransactionID
	settlement.IncomeTransactionID = incomeTransactionID
	return settlement, nil
}

func (repository *PostgresFinanceRepository) GetDriverBalance(ctx context.Context, driverID uuid.UUID) (domain.DriverBalance, error) {
	const query = `
		SELECT driver_id, available_balance_cents, pending_balance_cents, currency, updated_at
		FROM driver_balances
		WHERE driver_id = $1`

	var balance domain.DriverBalance
	var currency string
	if err := repository.pool.QueryRow(ctx, query, driverID).Scan(
		&balance.DriverID,
		&balance.AvailableBalance.Amount,
		&balance.PendingBalance.Amount,
		&currency,
		&balance.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DriverBalance{
				DriverID:         driverID,
				AvailableBalance: domain.Money{Currency: "RUB"},
				PendingBalance:   domain.Money{Currency: "RUB"},
			}, nil
		}
		return domain.DriverBalance{}, fmt.Errorf("select driver balance: %w", err)
	}
	balance.AvailableBalance.Currency = currency
	balance.PendingBalance.Currency = currency
	return balance, nil
}

func (repository *PostgresFinanceRepository) ListDriverTransactions(ctx context.Context, driverID uuid.UUID, limit int) ([]domain.FinancialTransaction, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT `+financialTransactionColumns("ft")+`
		FROM financial_transactions ft
		WHERE ft.driver_id = $1
		ORDER BY ft.created_at DESC
		LIMIT $2`, driverID, limit)
	if err != nil {
		return nil, fmt.Errorf("select driver financial transactions: %w", err)
	}
	defer rows.Close()

	return scanFinancialTransactions(rows)
}

func (repository *PostgresFinanceRepository) GetTaxiParkBalance(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error) {
	const query = `
		SELECT id, balance_cents, 'RUB'::text, updated_at
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL`

	var balance domain.TaxiParkBalance
	var currency string
	if err := repository.pool.QueryRow(ctx, query, ownerUserID).Scan(
		&balance.TaxiParkID,
		&balance.AvailableBalance.Amount,
		&currency,
		&balance.UpdatedAt,
	); err != nil {
		return domain.TaxiParkBalance{}, fmt.Errorf("select taxi park balance: %w", err)
	}
	balance.AvailableBalance.Currency = currency
	return balance, nil
}

func (repository *PostgresFinanceRepository) ListTaxiParkDrivers(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.TaxiParkDriver, error) {
	const query = `
		SELECT d.id,
		       d.user_id,
		       u.phone,
		       COALESCE(u.email, ''),
		       COALESCE(u.first_name, ''),
		       COALESCE(u.last_name, ''),
		       trim(coalesce(u.first_name, '') || ' ' || coalesce(u.last_name, '')) AS full_name,
		       d.status,
		       d.verification_status,
		       d.rating::float8,
		       d.ratings_count,
		       d.birth_date,
		       COALESCE(d.license_series, ''),
		       COALESCE(d.license_number, ''),
		       COALESCE(d.license_category, ''),
		       d.license_issued_at,
		       d.license_expires_at,
		       d.driving_experience_from,
		       d.has_no_taxi_work_restrictions,
		       d.federal_law_580_compliant,
		       d.regional_requirements_compliant,
		       d.medical_check_passed,
		       d.pretrip_control_required,
		       d.pretrip_control_passed,
		       d.no_transport_ban,
		       d.verification_checked_at,
		       d.verification_checked_by,
		       d.is_verified,
		       COALESCE(d.blocked_reason, ''),
		       COALESCE(d.taxi_park_comment, ''),
		       d.created_at,
		       d.updated_at
		FROM taxi_parks tp
		JOIN drivers d ON d.taxi_park_id = tp.id
		JOIN users u ON u.id = d.user_id
		WHERE tp.owner_user_id = $1 AND tp.deleted_at IS NULL AND d.deleted_at IS NULL
		ORDER BY d.created_at DESC
		LIMIT $2`

	rows, err := repository.pool.Query(ctx, query, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park drivers: %w", err)
	}
	defer rows.Close()

	drivers := make([]finance.TaxiParkDriver, 0, limit)
	for rows.Next() {
		var driver finance.TaxiParkDriver
		var birthDate pgtype.Date
		var licenseIssuedAt pgtype.Date
		var licenseExpiresAt pgtype.Date
		var drivingExperienceFrom pgtype.Date
		var verificationCheckedAt pgtype.Timestamptz
		var verificationCheckedBy pgtype.UUID
		if err := rows.Scan(
			&driver.ID,
			&driver.UserID,
			&driver.Phone,
			&driver.Email,
			&driver.FirstName,
			&driver.LastName,
			&driver.FullName,
			&driver.Status,
			&driver.VerificationStatus,
			&driver.Rating,
			&driver.RatingsCount,
			&birthDate,
			&driver.LicenseSeries,
			&driver.LicenseNumber,
			&driver.LicenseCategory,
			&licenseIssuedAt,
			&licenseExpiresAt,
			&drivingExperienceFrom,
			&driver.HasNoTaxiWorkRestrictions,
			&driver.FederalLaw580Compliant,
			&driver.RegionalRequirementsCompliant,
			&driver.MedicalCheckPassed,
			&driver.PretripControlRequired,
			&driver.PretripControlPassed,
			&driver.NoTransportBan,
			&verificationCheckedAt,
			&verificationCheckedBy,
			&driver.IsVerified,
			&driver.BlockedReason,
			&driver.TaxiParkComment,
			&driver.CreatedAt,
			&driver.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan taxi park driver: %w", err)
		}
		driver.BirthDate = pgDateTimePtr(birthDate)
		driver.LicenseIssuedAt = pgDateTimePtr(licenseIssuedAt)
		driver.LicenseExpiresAt = pgDateTimePtr(licenseExpiresAt)
		driver.DrivingExperienceFrom = pgDateTimePtr(drivingExperienceFrom)
		if verificationCheckedAt.Valid {
			driver.VerificationCheckedAt = &verificationCheckedAt.Time
		}
		if verificationCheckedBy.Valid {
			value, err := uuid.FromBytes(verificationCheckedBy.Bytes[:])
			if err != nil {
				return nil, fmt.Errorf("parse taxi park driver verifier id: %w", err)
			}
			driver.VerificationCheckedBy = &value
		}
		drivers = append(drivers, driver)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park drivers: %w", err)
	}
	return drivers, nil
}

func (repository *PostgresFinanceRepository) ListTaxiParkOrders(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.TaxiParkOrder, error) {
	const query = `
		SELECT
			o.id,
			o.driver_id,
			o.status,
			COALESCE((COALESCE(o.final_price, o.estimated_price) * 100)::bigint, 0),
			'RUB'::text,
			o.created_at,
			o.completed_at
		FROM taxi_parks tp
		JOIN orders o ON (
			o.metadata->>'taxi_park_id' = tp.id::text
			OR EXISTS (
				SELECT 1
				FROM drivers d
				WHERE d.id = o.driver_id
				  AND d.taxi_park_id = tp.id
				  AND d.deleted_at IS NULL
			)
		)
		WHERE tp.owner_user_id = $1
		  AND tp.deleted_at IS NULL
		  AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC
		LIMIT $2`

	rows, err := repository.pool.Query(ctx, query, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park orders: %w", err)
	}
	defer rows.Close()

	orders := make([]finance.TaxiParkOrder, 0, limit)
	for rows.Next() {
		var order finance.TaxiParkOrder
		var driverID pgtype.UUID
		var currency string
		var completedAt pgtype.Timestamptz
		if err := rows.Scan(&order.ID, &driverID, &order.Status, &order.GrossAmount.Amount, &currency, &order.CreatedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("scan taxi park order: %w", err)
		}
		if driverID.Valid {
			value := uuid.UUID(driverID.Bytes)
			order.DriverID = &value
		}
		order.GrossAmount.Currency = currency
		if completedAt.Valid {
			order.CompletedAt = &completedAt.Time
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park orders: %w", err)
	}
	return orders, nil
}

func (repository *PostgresFinanceRepository) ListTaxiParkTransactions(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]domain.FinancialTransaction, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT `+financialTransactionColumns("ft")+`
		FROM financial_transactions ft
		JOIN taxi_parks tp ON tp.id = ft.taxi_park_id
		WHERE tp.owner_user_id = $1
		ORDER BY ft.created_at DESC
		LIMIT $2`, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park financial transactions: %w", err)
	}
	defer rows.Close()

	return scanFinancialTransactions(rows)
}

func (repository *PostgresFinanceRepository) GetAdminOverview(ctx context.Context, periodFrom time.Time, periodTo time.Time) (finance.AdminOverview, error) {
	const query = `
		SELECT
			COALESCE(SUM(gross_amount_cents) FILTER (WHERE transaction_type = 'driver_income'), 0),
			COALESCE(SUM(commission_amount_cents) FILTER (WHERE transaction_type = 'driver_income'), 0),
			COALESCE(SUM(net_amount_cents) FILTER (WHERE transaction_type = 'driver_income' AND taxi_park_id IS NULL), 0),
			COALESCE(SUM(net_amount_cents) FILTER (WHERE transaction_type = 'driver_income' AND taxi_park_id IS NOT NULL), 0),
			COUNT(*) FILTER (WHERE transaction_type = 'driver_income')
		FROM financial_transactions
		WHERE created_at >= $1 AND created_at < $2`

	var overview finance.AdminOverview
	overview.PeriodFrom = periodFrom
	overview.PeriodTo = periodTo
	var completedOrdersRevenue int64
	var totalCommissions int64
	var driverPayouts int64
	var taxiParkRevenue int64
	if err := repository.pool.QueryRow(ctx, query, periodFrom, periodTo).Scan(
		&completedOrdersRevenue,
		&totalCommissions,
		&driverPayouts,
		&taxiParkRevenue,
		&overview.CompletedOrdersCount,
	); err != nil {
		return finance.AdminOverview{}, fmt.Errorf("select admin finance overview: %w", err)
	}

	overview.CompletedOrdersRevenue = domain.Money{Amount: completedOrdersRevenue, Currency: "RUB"}
	overview.TotalCommissions = domain.Money{Amount: totalCommissions, Currency: "RUB"}
	overview.DriverPayouts = domain.Money{Amount: driverPayouts, Currency: "RUB"}
	overview.TaxiParkRevenue = domain.Money{Amount: taxiParkRevenue, Currency: "RUB"}
	if overview.CompletedOrdersCount > 0 {
		overview.AverageCommission = domain.Money{Amount: totalCommissions / overview.CompletedOrdersCount, Currency: "RUB"}
	} else {
		overview.AverageCommission = domain.Money{Currency: "RUB"}
	}

	return overview, nil
}

func insertFinancialTransaction(ctx context.Context, tx pgx.Tx, settlement domain.OrderSettlement, transactionType domain.TransactionType) (uuid.UUID, error) {
	const query = `
		INSERT INTO financial_transactions (
			order_id,
			driver_id,
			taxi_park_id,
			transaction_type,
			gross_amount_cents,
			commission_percent,
			commission_amount_cents,
			net_amount_cents,
			currency,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6::numeric / 100, $7, $8, $9, $10)
		RETURNING id`

	netAmount := settlement.NetAmount.Amount
	if transactionType == domain.TransactionTypeCommission {
		netAmount = settlement.CommissionAmount.Amount
	}
	metadata, err := json.Marshal(map[string]any{
		"commission_source": settlement.CommissionRate.Source,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal finance metadata: %w", err)
	}

	var transactionID uuid.UUID
	if err := tx.QueryRow(
		ctx,
		query,
		settlement.OrderID,
		settlement.DriverID,
		settlement.TaxiParkID,
		transactionType,
		settlement.GrossAmount.Amount,
		settlement.CommissionRate.BasisPoints,
		settlement.CommissionAmount.Amount,
		netAmount,
		settlement.GrossAmount.Currency,
		metadata,
	).Scan(&transactionID); err != nil {
		return uuid.Nil, fmt.Errorf("insert financial transaction: %w", err)
	}

	return transactionID, nil
}

func insertFinanceAudit(ctx context.Context, tx pgx.Tx, transactionID uuid.UUID, settlement domain.OrderSettlement, eventType string) error {
	payload, err := json.Marshal(map[string]any{
		"gross_amount_cents":      settlement.GrossAmount.Amount,
		"commission_bps":          settlement.CommissionRate.BasisPoints,
		"commission_amount_cents": settlement.CommissionAmount.Amount,
		"net_amount_cents":        settlement.NetAmount.Amount,
		"taxi_park_id":            settlement.TaxiParkID,
	})
	if err != nil {
		return fmt.Errorf("marshal finance audit payload: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO finance_audit_events (transaction_id, order_id, event_type, payload)
		VALUES ($1, $2, $3, $4)`, transactionID, settlement.OrderID, eventType, payload); err != nil {
		return fmt.Errorf("insert finance audit event: %w", err)
	}
	return nil
}

func financialTransactionColumns(alias string) string {
	return fmt.Sprintf(`
	%s.id,
	%s.order_id,
	%s.driver_id,
	%s.taxi_park_id,
	%s.transaction_type,
	%s.gross_amount_cents,
	(%s.commission_percent * 100)::integer,
	%s.commission_amount_cents,
	%s.net_amount_cents,
	%s.currency,
	%s.created_at`, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias)
}

func pgDateTimePtr(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func scanFinancialTransactions(rows pgx.Rows) ([]domain.FinancialTransaction, error) {
	transactions := make([]domain.FinancialTransaction, 0)
	for rows.Next() {
		transaction, err := scanFinancialTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate financial transactions: %w", err)
	}
	return transactions, nil
}

func scanFinancialTransaction(row pgx.Row) (domain.FinancialTransaction, error) {
	var transaction domain.FinancialTransaction
	var orderID pgtype.UUID
	var driverID pgtype.UUID
	var taxiParkID pgtype.UUID
	var currency string
	if err := row.Scan(
		&transaction.ID,
		&orderID,
		&driverID,
		&taxiParkID,
		&transaction.TransactionType,
		&transaction.GrossAmount.Amount,
		&transaction.CommissionBasisPoints,
		&transaction.CommissionAmount.Amount,
		&transaction.NetAmount.Amount,
		&currency,
		&transaction.CreatedAt,
	); err != nil {
		return domain.FinancialTransaction{}, fmt.Errorf("scan financial transaction: %w", err)
	}
	if orderID.Valid {
		value := uuid.UUID(orderID.Bytes)
		transaction.OrderID = &value
	}
	if driverID.Valid {
		value := uuid.UUID(driverID.Bytes)
		transaction.DriverID = &value
	}
	if taxiParkID.Valid {
		value := uuid.UUID(taxiParkID.Bytes)
		transaction.TaxiParkID = &value
	}
	transaction.Currency = currency
	transaction.GrossAmount.Currency = currency
	transaction.CommissionAmount.Currency = currency
	transaction.NetAmount.Currency = currency
	return transaction, nil
}

func optionalInt32(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func isUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505"
}
