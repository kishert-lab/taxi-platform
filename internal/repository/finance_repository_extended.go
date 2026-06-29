package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/finance"
)

func (repository *PostgresFinanceRepository) ListDriverOrderFinances(ctx context.Context, userID uuid.UUID, limit int) ([]finance.OrderFinance, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT oft.id, oft.order_id, oft.taxi_park_id, oft.driver_id, oft.passenger_id,
		       (oft.order_total_amount * 100)::bigint,
		       (oft.driver_commission_percent * 100)::integer,
		       (oft.taxi_park_commission_amount * 100)::bigint,
		       (oft.driver_income_amount * 100)::bigint,
		       (oft.platform_service_fee_percent * 100)::integer,
		       (oft.platform_service_fee_amount * 100)::bigint,
		       (oft.taxi_park_income_amount * 100)::bigint,
		       oft.currency, oft.status, oft.created_at, oft.updated_at
		FROM order_financial_transactions oft
		JOIN drivers d ON d.id = oft.driver_id
		WHERE d.user_id = $1 AND d.deleted_at IS NULL
		ORDER BY oft.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select driver order finances: %w", err)
	}
	defer rows.Close()
	return scanOrderFinances(rows)
}

func (repository *PostgresFinanceRepository) ListDriverPayouts(ctx context.Context, userID uuid.UUID, limit int) ([]finance.DriverPayout, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT p.id, p.driver_id, p.taxi_park_id, (p.amount * 100)::bigint, p.currency, p.status,
		       p.period_from, p.period_to, COALESCE(p.payment_method, ''), COALESCE(p.payment_document_number, ''),
		       COALESCE(p.comment, ''), p.created_by, p.created_at, p.paid_at, p.updated_at
		FROM driver_payouts p
		JOIN drivers d ON d.id = p.driver_id
		WHERE d.user_id = $1 AND d.deleted_at IS NULL
		ORDER BY p.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select driver payouts: %w", err)
	}
	defer rows.Close()
	return scanDriverPayouts(rows)
}

func (repository *PostgresFinanceRepository) ListDriverFinanceDocuments(ctx context.Context, userID uuid.UUID, limit int) ([]finance.FinanceDocument, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT fd.id, fd.taxi_park_id, fd.driver_id, fd.order_id, fd.payout_id, fd.invoice_id,
		       fd.type, fd.number, fd.status, COALESCE(fd.file_url, ''), COALESCE(fd.payload::text, '{}'),
		       fd.created_at, fd.updated_at
		FROM finance_documents fd
		JOIN drivers d ON d.id = fd.driver_id
		WHERE d.user_id = $1 AND d.deleted_at IS NULL
		ORDER BY fd.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select driver documents: %w", err)
	}
	defer rows.Close()
	return scanFinanceDocuments(rows)
}

func (repository *PostgresFinanceRepository) GetTaxiParkOverview(ctx context.Context, ownerUserID uuid.UUID, periodFrom time.Time, periodTo time.Time) (finance.TaxiParkFinanceOverview, error) {
	taxiParkID, err := repository.resolveTaxiParkIDByOwner(ctx, ownerUserID)
	if err != nil {
		return finance.TaxiParkFinanceOverview{}, err
	}
	return repository.getTaxiParkOverviewByID(ctx, taxiParkID, periodFrom, periodTo)
}

func (repository *PostgresFinanceRepository) ListTaxiParkOrderFinances(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.OrderFinance, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT oft.id, oft.order_id, oft.taxi_park_id, oft.driver_id, oft.passenger_id,
		       (oft.order_total_amount * 100)::bigint,
		       (oft.driver_commission_percent * 100)::integer,
		       (oft.taxi_park_commission_amount * 100)::bigint,
		       (oft.driver_income_amount * 100)::bigint,
		       (oft.platform_service_fee_percent * 100)::integer,
		       (oft.platform_service_fee_amount * 100)::bigint,
		       (oft.taxi_park_income_amount * 100)::bigint,
		       oft.currency, oft.status, oft.created_at, oft.updated_at
		FROM order_financial_transactions oft
		JOIN taxi_parks tp ON tp.id = oft.taxi_park_id
		WHERE tp.owner_user_id = $1 AND tp.deleted_at IS NULL
		ORDER BY oft.created_at DESC
		LIMIT $2`, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park order finances: %w", err)
	}
	defer rows.Close()
	return scanOrderFinances(rows)
}

func (repository *PostgresFinanceRepository) GetTaxiParkDriverBalance(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) (domain.DriverBalance, error) {
	const query = `
		SELECT db.driver_id, db.available_balance_cents, db.pending_balance_cents, db.currency, db.updated_at
		FROM driver_balances db
		JOIN drivers d ON d.id = db.driver_id
		JOIN taxi_parks tp ON tp.id = d.taxi_park_id
		WHERE db.driver_id = $1 AND tp.owner_user_id = $2 AND d.deleted_at IS NULL AND tp.deleted_at IS NULL`

	var balance domain.DriverBalance
	var currency string
	if err := repository.pool.QueryRow(ctx, query, driverID, ownerUserID).Scan(
		&balance.DriverID,
		&balance.AvailableBalance.Amount,
		&balance.PendingBalance.Amount,
		&currency,
		&balance.UpdatedAt,
	); err != nil {
		return domain.DriverBalance{}, fmt.Errorf("select taxi park driver balance: %w", err)
	}
	balance.AvailableBalance.Currency = currency
	balance.PendingBalance.Currency = currency
	return balance, nil
}

func (repository *PostgresFinanceRepository) ListTaxiParkDriverPayouts(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, limit int) ([]finance.DriverPayout, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT p.id, p.driver_id, p.taxi_park_id, (p.amount * 100)::bigint, p.currency, p.status,
		       p.period_from, p.period_to, COALESCE(p.payment_method, ''), COALESCE(p.payment_document_number, ''),
		       COALESCE(p.comment, ''), p.created_by, p.created_at, p.paid_at, p.updated_at
		FROM driver_payouts p
		JOIN taxi_parks tp ON tp.id = p.taxi_park_id
		WHERE tp.owner_user_id = $1 AND p.driver_id = $2
		ORDER BY p.created_at DESC
		LIMIT $3`, ownerUserID, driverID, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park driver payouts: %w", err)
	}
	defer rows.Close()
	return scanDriverPayouts(rows)
}

func (repository *PostgresFinanceRepository) CreateDriverPayout(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, input finance.CreateDriverPayoutInput) (finance.DriverPayout, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return finance.DriverPayout{}, fmt.Errorf("begin driver payout transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	taxiParkID, err := repository.resolveTaxiParkIDByOwnerTx(ctx, tx, ownerUserID)
	if err != nil {
		return finance.DriverPayout{}, err
	}

	var currentBalance int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(db.available_balance_cents, 0)
		FROM drivers d
		LEFT JOIN driver_balances db ON db.driver_id = d.id
		WHERE d.id = $1 AND d.taxi_park_id = $2 AND d.deleted_at IS NULL`, driverID, taxiParkID).Scan(&currentBalance); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("select driver payout balance: %w", err)
	}
	if currentBalance < input.AmountCents {
		return finance.DriverPayout{}, finance.ErrInsufficientDriverBalance
	}

	var payout finance.DriverPayout
	if err := tx.QueryRow(ctx, `
		INSERT INTO driver_payouts (
			driver_id, taxi_park_id, amount, currency, status, period_from, period_to, comment, created_by
		) VALUES ($1, $2, $3::numeric, 'RUB', 'created', $4, $5, $6, $7)
		RETURNING id, driver_id, taxi_park_id, (amount * 100)::bigint, currency, status, period_from, period_to,
		          COALESCE(payment_method, ''), COALESCE(payment_document_number, ''), COALESCE(comment, ''),
		          created_by, created_at, paid_at, updated_at`,
		driverID, taxiParkID, moneyCentsToNumeric(input.AmountCents), input.PeriodFrom, input.PeriodTo, input.Comment, ownerUserID,
	).Scan(
		&payout.ID, &payout.DriverID, &payout.TaxiParkID, &payout.Amount.Amount, &payout.Amount.Currency, &payout.Status,
		nullableTime(&payout.PeriodFrom), nullableTime(&payout.PeriodTo), &payout.PaymentMethod, &payout.PaymentDocumentNumber,
		&payout.Comment, uuidPtrScanner(&payout.CreatedBy), &payout.CreatedAt, nullableTime(&payout.PaidAt), &payout.UpdatedAt,
	); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("insert driver payout: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("commit driver payout transaction: %w", err)
	}
	return payout, nil
}

func (repository *PostgresFinanceRepository) ApproveDriverPayout(ctx context.Context, ownerUserID uuid.UUID, payoutID uuid.UUID) (finance.DriverPayout, error) {
	return repository.updateDriverPayoutStatus(ctx, ownerUserID, payoutID, "approved", false)
}

func (repository *PostgresFinanceRepository) MarkDriverPayoutPaid(ctx context.Context, ownerUserID uuid.UUID, payoutID uuid.UUID) (finance.DriverPayout, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return finance.DriverPayout{}, fmt.Errorf("begin payout paid transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	payout, err := repository.updateDriverPayoutStatusTx(ctx, tx, ownerUserID, payoutID, "paid", true)
	if err != nil {
		return finance.DriverPayout{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE driver_balances
		SET available_balance_cents = available_balance_cents - $2,
		    version = version + 1
		WHERE driver_id = $1`, payout.DriverID, payout.Amount.Amount); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("update driver balance after payout: %w", err)
	}

	balanceAfter, err := nextLedgerBalance(ctx, tx, "driver_balance_ledger", "driver_id", payout.DriverID, payout.Amount.Amount, false)
	if err != nil {
		return finance.DriverPayout{}, fmt.Errorf("resolve driver payout ledger balance: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO driver_balance_ledger (
			driver_id, taxi_park_id, transaction_id, type, amount, currency, direction, balance_after, comment
		) VALUES ($1, $2, NULL, 'payout', $3::numeric, $4, 'debit', $5::numeric, $6)`,
		payout.DriverID, payout.TaxiParkID, moneyCentsToNumeric(payout.Amount.Amount), payout.Amount.Currency, moneyCentsToNumeric(balanceAfter), payout.Comment,
	); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("insert driver payout ledger: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("commit payout paid transaction: %w", err)
	}
	return payout, nil
}

func (repository *PostgresFinanceRepository) GetTaxiParkPlatformFeeDebt(ctx context.Context, ownerUserID uuid.UUID) (domain.Money, error) {
	taxiParkID, err := repository.resolveTaxiParkIDByOwner(ctx, ownerUserID)
	if err != nil {
		return domain.Money{}, err
	}
	return repository.getTaxiParkPlatformFeeDebtByID(ctx, taxiParkID)
}

func (repository *PostgresFinanceRepository) ListTaxiParkPlatformFeeAccruals(ctx context.Context, ownerUserID uuid.UUID, periodFrom time.Time, periodTo time.Time, limit int) ([]finance.OrderFinance, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT oft.id, oft.order_id, oft.taxi_park_id, oft.driver_id, oft.passenger_id,
		       (oft.order_total_amount * 100)::bigint,
		       (oft.driver_commission_percent * 100)::integer,
		       (oft.taxi_park_commission_amount * 100)::bigint,
		       (oft.driver_income_amount * 100)::bigint,
		       (oft.platform_service_fee_percent * 100)::integer,
		       (oft.platform_service_fee_amount * 100)::bigint,
		       (oft.taxi_park_income_amount * 100)::bigint,
		       oft.currency, oft.status, oft.created_at, oft.updated_at
		FROM order_financial_transactions oft
		JOIN taxi_parks tp ON tp.id = oft.taxi_park_id
		WHERE tp.owner_user_id = $1 AND oft.created_at >= $2 AND oft.created_at < $3
		ORDER BY oft.created_at DESC
		LIMIT $4`, ownerUserID, periodFrom, periodTo, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park platform fee accruals: %w", err)
	}
	defer rows.Close()
	return scanOrderFinances(rows)
}

func (repository *PostgresFinanceRepository) ListTaxiParkPlatformInvoices(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.PlatformInvoice, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT pi.id, pi.taxi_park_id, (pi.amount * 100)::bigint, pi.currency, pi.period_from, pi.period_to, pi.status,
		       pi.invoice_number, COALESCE(pi.document_url, ''), pi.created_at, pi.paid_at, pi.updated_at
		FROM platform_invoices pi
		JOIN taxi_parks tp ON tp.id = pi.taxi_park_id
		WHERE tp.owner_user_id = $1
		ORDER BY pi.created_at DESC
		LIMIT $2`, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park platform invoices: %w", err)
	}
	defer rows.Close()
	return scanPlatformInvoices(rows)
}

func (repository *PostgresFinanceRepository) CreateTaxiParkPlatformInvoice(ctx context.Context, ownerUserID uuid.UUID, input finance.CreatePlatformInvoiceInput) (finance.PlatformInvoice, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("begin platform invoice transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	taxiParkID, err := repository.resolveTaxiParkIDByOwnerTx(ctx, tx, ownerUserID)
	if err != nil {
		return finance.PlatformInvoice{}, err
	}

	var amountCents int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM((amount * 100)::bigint), 0)
		FROM taxi_park_platform_fee_ledger
		WHERE taxi_park_id = $1
		  AND invoice_id IS NULL
		  AND type = 'platform_service_fee_accrual'
		  AND created_at >= $2 AND created_at < $3`, taxiParkID, input.PeriodFrom, input.PeriodTo).Scan(&amountCents); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("sum platform invoice accruals: %w", err)
	}

	invoiceNumber := fmt.Sprintf("INV-%s-%d", taxiParkID.String()[:8], time.Now().UTC().Unix())
	var invoice finance.PlatformInvoice
	if err := tx.QueryRow(ctx, `
		INSERT INTO platform_invoices (
			taxi_park_id, amount, currency, period_from, period_to, status, invoice_number
		) VALUES ($1, $2::numeric, 'RUB', $3, $4, 'issued', $5)
		RETURNING id, taxi_park_id, (amount * 100)::bigint, currency, period_from, period_to, status,
		          invoice_number, COALESCE(document_url, ''), created_at, paid_at, updated_at`,
		taxiParkID, moneyCentsToNumeric(amountCents), input.PeriodFrom, input.PeriodTo, invoiceNumber,
	).Scan(
		&invoice.ID, &invoice.TaxiParkID, &invoice.Amount.Amount, &invoice.Amount.Currency, &invoice.PeriodFrom, &invoice.PeriodTo,
		&invoice.Status, &invoice.InvoiceNumber, &invoice.DocumentURL, &invoice.CreatedAt, nullableTime(&invoice.PaidAt), &invoice.UpdatedAt,
	); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("insert platform invoice: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE taxi_park_platform_fee_ledger
		SET invoice_id = $4
		WHERE taxi_park_id = $1
		  AND invoice_id IS NULL
		  AND type = 'platform_service_fee_accrual'
		  AND created_at >= $2 AND created_at < $3`, taxiParkID, input.PeriodFrom, input.PeriodTo, invoice.ID); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("attach invoice to fee accruals: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("commit platform invoice transaction: %w", err)
	}
	return invoice, nil
}

func (repository *PostgresFinanceRepository) ListTaxiParkDocuments(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.FinanceDocument, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT fd.id, fd.taxi_park_id, fd.driver_id, fd.order_id, fd.payout_id, fd.invoice_id,
		       fd.type, fd.number, fd.status, COALESCE(fd.file_url, ''), COALESCE(fd.payload::text, '{}'),
		       fd.created_at, fd.updated_at
		FROM finance_documents fd
		JOIN taxi_parks tp ON tp.id = fd.taxi_park_id
		WHERE tp.owner_user_id = $1
		ORDER BY fd.created_at DESC
		LIMIT $2`, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("select taxi park documents: %w", err)
	}
	defer rows.Close()
	return scanFinanceDocuments(rows)
}

func (repository *PostgresFinanceRepository) GetAdminTaxiParkOverview(ctx context.Context, taxiParkID uuid.UUID, periodFrom time.Time, periodTo time.Time) (finance.TaxiParkFinanceOverview, error) {
	return repository.getTaxiParkOverviewByID(ctx, taxiParkID, periodFrom, periodTo)
}

func (repository *PostgresFinanceRepository) GetAdminTaxiParkPlatformFeeDebt(ctx context.Context, taxiParkID uuid.UUID) (domain.Money, error) {
	return repository.getTaxiParkPlatformFeeDebtByID(ctx, taxiParkID)
}

func (repository *PostgresFinanceRepository) ListAdminPlatformInvoices(ctx context.Context, limit int) ([]finance.PlatformInvoice, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, taxi_park_id, (amount * 100)::bigint, currency, period_from, period_to, status,
		       invoice_number, COALESCE(document_url, ''), created_at, paid_at, updated_at
		FROM platform_invoices
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("select admin platform invoices: %w", err)
	}
	defer rows.Close()
	return scanPlatformInvoices(rows)
}

func (repository *PostgresFinanceRepository) MarkAdminPlatformInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (finance.PlatformInvoice, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("begin admin invoice paid transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	var invoice finance.PlatformInvoice
	if err := tx.QueryRow(ctx, `
		UPDATE platform_invoices
		SET status = 'paid', paid_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING id, taxi_park_id, (amount * 100)::bigint, currency, period_from, period_to, status,
		          invoice_number, COALESCE(document_url, ''), created_at, paid_at, updated_at`, invoiceID,
	).Scan(
		&invoice.ID, &invoice.TaxiParkID, &invoice.Amount.Amount, &invoice.Amount.Currency, &invoice.PeriodFrom, &invoice.PeriodTo,
		&invoice.Status, &invoice.InvoiceNumber, &invoice.DocumentURL, &invoice.CreatedAt, nullableTime(&invoice.PaidAt), &invoice.UpdatedAt,
	); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("update platform invoice paid: %w", err)
	}

	debtBalanceAfter, err := nextLedgerBalance(ctx, tx, "taxi_park_platform_fee_ledger", "taxi_park_id", invoice.TaxiParkID, invoice.Amount.Amount, false)
	if err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("resolve taxi park fee payment balance: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO taxi_park_platform_fee_ledger (
			taxi_park_id, invoice_id, type, amount, currency, direction, balance_after
		) VALUES ($1, $2, 'platform_service_fee_payment', $3::numeric, $4, 'credit', $5::numeric)`,
		invoice.TaxiParkID, invoice.ID, moneyCentsToNumeric(invoice.Amount.Amount), invoice.Amount.Currency, moneyCentsToNumeric(debtBalanceAfter),
	); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("insert taxi park fee payment ledger: %w", err)
	}

	platformBalanceAfter, err := nextLedgerBalance(ctx, tx, "platform_balance_ledger", "taxi_park_id", invoice.TaxiParkID, invoice.Amount.Amount, true)
	if err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("resolve platform payment received balance: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO platform_balance_ledger (
			taxi_park_id, invoice_id, type, amount, currency, direction, balance_after
		) VALUES ($1, $2, 'service_fee_payment_received', $3::numeric, $4, 'credit', $5::numeric)`,
		invoice.TaxiParkID, invoice.ID, moneyCentsToNumeric(invoice.Amount.Amount), invoice.Amount.Currency, moneyCentsToNumeric(platformBalanceAfter),
	); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("insert platform payment received ledger: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return finance.PlatformInvoice{}, fmt.Errorf("commit admin invoice paid transaction: %w", err)
	}
	return invoice, nil
}

func (repository *PostgresFinanceRepository) ListAdminDocuments(ctx context.Context, limit int) ([]finance.FinanceDocument, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, taxi_park_id, driver_id, order_id, payout_id, invoice_id,
		       type, number, status, COALESCE(file_url, ''), COALESCE(payload::text, '{}'),
		       created_at, updated_at
		FROM finance_documents
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("select admin finance documents: %w", err)
	}
	defer rows.Close()
	return scanFinanceDocuments(rows)
}

func (repository *PostgresFinanceRepository) GenerateDriverPayoutAct(ctx context.Context, payoutID uuid.UUID) (finance.FinanceDocument, error) {
	return repository.insertFinanceDocument(ctx, financeDocumentInsertInput{
		documentType: "driver_payout_act",
		numberPrefix: "DPA",
		buildPayload: func() (map[string]any, linkedFinanceDocumentRefs, error) {
			var refs linkedFinanceDocumentRefs
			var amount string
			if err := repository.pool.QueryRow(ctx, `
				SELECT p.taxi_park_id, p.driver_id, p.id, p.amount::text
				FROM driver_payouts p
				WHERE p.id = $1`, payoutID).Scan(&refs.taxiParkID, &refs.driverID, &refs.payoutID, &amount); err != nil {
				return nil, refs, fmt.Errorf("select driver payout for document: %w", err)
			}
			return map[string]any{"payout_id": payoutID, "amount": amount}, refs, nil
		},
	})
}

func (repository *PostgresFinanceRepository) GeneratePlatformInvoice(ctx context.Context, invoiceID uuid.UUID) (finance.FinanceDocument, error) {
	return repository.insertFinanceDocument(ctx, financeDocumentInsertInput{
		documentType: "taxi_park_platform_invoice",
		numberPrefix: "TPI",
		buildPayload: func() (map[string]any, linkedFinanceDocumentRefs, error) {
			var refs linkedFinanceDocumentRefs
			var amount string
			if err := repository.pool.QueryRow(ctx, `
				SELECT taxi_park_id, id, amount::text, invoice_number
				FROM platform_invoices
				WHERE id = $1`, invoiceID).Scan(&refs.taxiParkID, &refs.invoiceID, &amount, &refs.number); err != nil {
				return nil, refs, fmt.Errorf("select invoice for document: %w", err)
			}
			return map[string]any{"invoice_id": invoiceID, "amount": amount}, refs, nil
		},
	})
}

func (repository *PostgresFinanceRepository) GeneratePlatformAct(ctx context.Context, invoiceID uuid.UUID) (finance.FinanceDocument, error) {
	return repository.insertFinanceDocument(ctx, financeDocumentInsertInput{
		documentType: "taxi_park_platform_act",
		numberPrefix: "TPA",
		buildPayload: func() (map[string]any, linkedFinanceDocumentRefs, error) {
			var refs linkedFinanceDocumentRefs
			var status string
			if err := repository.pool.QueryRow(ctx, `
				SELECT taxi_park_id, id, status, invoice_number
				FROM platform_invoices
				WHERE id = $1`, invoiceID).Scan(&refs.taxiParkID, &refs.invoiceID, &status, &refs.number); err != nil {
				return nil, refs, fmt.Errorf("select invoice for act: %w", err)
			}
			return map[string]any{"invoice_id": invoiceID, "status": status}, refs, nil
		},
	})
}

func (repository *PostgresFinanceRepository) GenerateOrderFinancialReport(ctx context.Context, orderID uuid.UUID) (finance.FinanceDocument, error) {
	return repository.insertFinanceDocument(ctx, financeDocumentInsertInput{
		documentType: "order_financial_report",
		numberPrefix: "OFR",
		buildPayload: func() (map[string]any, linkedFinanceDocumentRefs, error) {
			var refs linkedFinanceDocumentRefs
			var orderTotal string
			if err := repository.pool.QueryRow(ctx, `
				SELECT taxi_park_id, driver_id, order_id, order_total_amount::text
				FROM order_financial_transactions
				WHERE order_id = $1`, orderID).Scan(&refs.taxiParkID, &refs.driverID, &refs.orderID, &orderTotal); err != nil {
				return nil, refs, fmt.Errorf("select order financial report source: %w", err)
			}
			return map[string]any{"order_id": orderID, "order_total_amount": orderTotal}, refs, nil
		},
	})
}

func (repository *PostgresFinanceRepository) GenerateReconciliationAct(ctx context.Context, taxiParkID uuid.UUID, periodFrom time.Time, periodTo time.Time) (finance.FinanceDocument, error) {
	return repository.insertFinanceDocument(ctx, financeDocumentInsertInput{
		documentType: "reconciliation_act",
		numberPrefix: "RCA",
		buildPayload: func() (map[string]any, linkedFinanceDocumentRefs, error) {
			overview, err := repository.getTaxiParkOverviewByID(ctx, taxiParkID, periodFrom, periodTo)
			if err != nil {
				return nil, linkedFinanceDocumentRefs{}, err
			}
			refs := linkedFinanceDocumentRefs{taxiParkID: &taxiParkID}
			return map[string]any{
				"taxi_park_id": taxiParkID,
				"period_from":  periodFrom,
				"period_to":    periodTo,
				"orders_count": overview.OrdersCount,
			}, refs, nil
		},
	})
}

func (repository *PostgresFinanceRepository) getTaxiParkOverviewByID(ctx context.Context, taxiParkID uuid.UUID, periodFrom time.Time, periodTo time.Time) (finance.TaxiParkFinanceOverview, error) {
	var overview finance.TaxiParkFinanceOverview
	overview.TaxiParkID = taxiParkID
	overview.PeriodFrom = periodFrom
	overview.PeriodTo = periodTo
	if err := repository.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM((order_total_amount * 100)::bigint), 0),
			COALESCE(SUM((driver_income_amount * 100)::bigint), 0),
			COALESCE(SUM((taxi_park_commission_amount * 100)::bigint), 0),
			COALESCE(SUM((taxi_park_income_amount * 100)::bigint), 0),
			COALESCE(SUM((platform_service_fee_amount * 100)::bigint), 0)
		FROM order_financial_transactions
		WHERE taxi_park_id = $1 AND created_at >= $2 AND created_at < $3`, taxiParkID, periodFrom, periodTo,
	).Scan(
		&overview.OrdersCount,
		&overview.OrderTotalAmount.Amount,
		&overview.DriverIncomeAmount.Amount,
		&overview.TaxiParkCommissionAmount.Amount,
		&overview.TaxiParkIncomeAmount.Amount,
		&overview.PlatformServiceFeeAmount.Amount,
	); err != nil {
		return finance.TaxiParkFinanceOverview{}, fmt.Errorf("select taxi park overview: %w", err)
	}
	overview.OrderTotalAmount.Currency = "RUB"
	overview.DriverIncomeAmount.Currency = "RUB"
	overview.TaxiParkCommissionAmount.Currency = "RUB"
	overview.TaxiParkIncomeAmount.Currency = "RUB"
	overview.PlatformServiceFeeAmount.Currency = "RUB"
	debt, err := repository.getTaxiParkPlatformFeeDebtByID(ctx, taxiParkID)
	if err != nil {
		return finance.TaxiParkFinanceOverview{}, err
	}
	overview.PlatformDebtAmount = debt
	return overview, nil
}

func (repository *PostgresFinanceRepository) getTaxiParkPlatformFeeDebtByID(ctx context.Context, taxiParkID uuid.UUID) (domain.Money, error) {
	var amountCents int64
	if err := repository.pool.QueryRow(ctx, `
		SELECT COALESCE((balance_after * 100)::bigint, 0)
		FROM taxi_park_platform_fee_ledger
		WHERE taxi_park_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, taxiParkID).Scan(&amountCents); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.Money{}, fmt.Errorf("select taxi park platform fee debt: %w", err)
	}
	return domain.Money{Amount: amountCents, Currency: "RUB"}, nil
}

func (repository *PostgresFinanceRepository) updateDriverPayoutStatus(ctx context.Context, ownerUserID uuid.UUID, payoutID uuid.UUID, status string, setPaidAt bool) (finance.DriverPayout, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return finance.DriverPayout{}, fmt.Errorf("begin driver payout status transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)
	payout, err := repository.updateDriverPayoutStatusTx(ctx, tx, ownerUserID, payoutID, status, setPaidAt)
	if err != nil {
		return finance.DriverPayout{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("commit driver payout status transaction: %w", err)
	}
	return payout, nil
}

func (repository *PostgresFinanceRepository) updateDriverPayoutStatusTx(ctx context.Context, tx pgx.Tx, ownerUserID uuid.UUID, payoutID uuid.UUID, status string, setPaidAt bool) (finance.DriverPayout, error) {
	paidAtClause := "NULL"
	if setPaidAt {
		paidAtClause = "now()"
	}
	query := fmt.Sprintf(`
		UPDATE driver_payouts p
		SET status = $2, paid_at = %s, updated_at = now()
		FROM taxi_parks tp
		WHERE p.id = $1 AND tp.id = p.taxi_park_id AND tp.owner_user_id = $3
		RETURNING p.id, p.driver_id, p.taxi_park_id, (p.amount * 100)::bigint, p.currency, p.status,
		          p.period_from, p.period_to, COALESCE(p.payment_method, ''), COALESCE(p.payment_document_number, ''),
		          COALESCE(p.comment, ''), p.created_by, p.created_at, p.paid_at, p.updated_at`, paidAtClause)
	var payout finance.DriverPayout
	if err := tx.QueryRow(ctx, query, payoutID, status, ownerUserID).Scan(
		&payout.ID, &payout.DriverID, &payout.TaxiParkID, &payout.Amount.Amount, &payout.Amount.Currency, &payout.Status,
		nullableTime(&payout.PeriodFrom), nullableTime(&payout.PeriodTo), &payout.PaymentMethod, &payout.PaymentDocumentNumber,
		&payout.Comment, uuidPtrScanner(&payout.CreatedBy), &payout.CreatedAt, nullableTime(&payout.PaidAt), &payout.UpdatedAt,
	); err != nil {
		return finance.DriverPayout{}, fmt.Errorf("update driver payout status: %w", err)
	}
	return payout, nil
}

func (repository *PostgresFinanceRepository) resolveTaxiParkIDByOwner(ctx context.Context, ownerUserID uuid.UUID) (uuid.UUID, error) {
	return repository.resolveTaxiParkIDByOwnerTx(ctx, repository.pool, ownerUserID)
}

func (repository *PostgresFinanceRepository) resolveTaxiParkIDByOwnerTx(ctx context.Context, queryable interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, ownerUserID uuid.UUID) (uuid.UUID, error) {
	var taxiParkID uuid.UUID
	if err := queryable.QueryRow(ctx, `
		SELECT id
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL`, ownerUserID).Scan(&taxiParkID); err != nil {
		return uuid.Nil, fmt.Errorf("resolve taxi park by owner: %w", err)
	}
	return taxiParkID, nil
}

type linkedFinanceDocumentRefs struct {
	taxiParkID *uuid.UUID
	driverID   *uuid.UUID
	orderID    *uuid.UUID
	payoutID   *uuid.UUID
	invoiceID  *uuid.UUID
	number     string
}

type financeDocumentInsertInput struct {
	documentType string
	numberPrefix string
	buildPayload func() (map[string]any, linkedFinanceDocumentRefs, error)
}

func (repository *PostgresFinanceRepository) insertFinanceDocument(ctx context.Context, input financeDocumentInsertInput) (finance.FinanceDocument, error) {
	payload, refs, err := input.buildPayload()
	if err != nil {
		return finance.FinanceDocument{}, err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return finance.FinanceDocument{}, fmt.Errorf("marshal finance document payload: %w", err)
	}
	number := refs.number
	if number == "" {
		number = fmt.Sprintf("%s-%d", input.numberPrefix, time.Now().UTC().Unix())
	}

	var document finance.FinanceDocument
	var payloadText string
	if err := repository.pool.QueryRow(ctx, `
		INSERT INTO finance_documents (
			taxi_park_id, driver_id, order_id, payout_id, invoice_id, type, number, status, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'created', $8::jsonb)
		RETURNING id, taxi_park_id, driver_id, order_id, payout_id, invoice_id, type, number, status,
		          COALESCE(file_url, ''), COALESCE(payload::text, '{}'), created_at, updated_at`,
		refs.taxiParkID, refs.driverID, refs.orderID, refs.payoutID, refs.invoiceID, input.documentType, number, string(payloadBytes),
	).Scan(
		&document.ID, uuidPtrScanner(&document.TaxiParkID), uuidPtrScanner(&document.DriverID), uuidPtrScanner(&document.OrderID),
		uuidPtrScanner(&document.PayoutID), uuidPtrScanner(&document.InvoiceID), &document.Type, &document.Number, &document.Status,
		&document.FileURL, &payloadText, &document.CreatedAt, &document.UpdatedAt,
	); err != nil {
		return finance.FinanceDocument{}, fmt.Errorf("insert finance document: %w", err)
	}
	document.Payload = []byte(payloadText)
	return document, nil
}

func scanOrderFinances(rows pgx.Rows) ([]finance.OrderFinance, error) {
	items := make([]finance.OrderFinance, 0)
	for rows.Next() {
		var item finance.OrderFinance
		var taxiParkID pgtype.UUID
		var passengerID pgtype.UUID
		var currency string
		if err := rows.Scan(
			&item.ID, &item.OrderID, &taxiParkID, &item.DriverID, &passengerID,
			&item.OrderTotalAmount.Amount, &item.DriverCommissionBasisPoints, &item.TaxiParkCommissionAmount.Amount,
			&item.DriverIncomeAmount.Amount, &item.PlatformFeeBasisPoints, &item.PlatformFeeAmount.Amount,
			&item.TaxiParkIncomeAmount.Amount, &currency, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order finance: %w", err)
		}
		if taxiParkID.Valid {
			value := uuid.UUID(taxiParkID.Bytes)
			item.TaxiParkID = &value
		}
		if passengerID.Valid {
			value := uuid.UUID(passengerID.Bytes)
			item.PassengerID = &value
		}
		item.OrderTotalAmount.Currency = currency
		item.TaxiParkCommissionAmount.Currency = currency
		item.DriverIncomeAmount.Currency = currency
		item.PlatformFeeAmount.Currency = currency
		item.TaxiParkIncomeAmount.Currency = currency
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order finances: %w", err)
	}
	return items, nil
}

func scanDriverPayouts(rows pgx.Rows) ([]finance.DriverPayout, error) {
	items := make([]finance.DriverPayout, 0)
	for rows.Next() {
		var item finance.DriverPayout
		if err := rows.Scan(
			&item.ID, &item.DriverID, &item.TaxiParkID, &item.Amount.Amount, &item.Amount.Currency, &item.Status,
			nullableTime(&item.PeriodFrom), nullableTime(&item.PeriodTo), &item.PaymentMethod, &item.PaymentDocumentNumber,
			&item.Comment, uuidPtrScanner(&item.CreatedBy), &item.CreatedAt, nullableTime(&item.PaidAt), &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan driver payout: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver payouts: %w", err)
	}
	return items, nil
}

func scanPlatformInvoices(rows pgx.Rows) ([]finance.PlatformInvoice, error) {
	items := make([]finance.PlatformInvoice, 0)
	for rows.Next() {
		var item finance.PlatformInvoice
		if err := rows.Scan(
			&item.ID, &item.TaxiParkID, &item.Amount.Amount, &item.Amount.Currency, &item.PeriodFrom, &item.PeriodTo,
			&item.Status, &item.InvoiceNumber, &item.DocumentURL, &item.CreatedAt, nullableTime(&item.PaidAt), &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan platform invoice: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate platform invoices: %w", err)
	}
	return items, nil
}

func scanFinanceDocuments(rows pgx.Rows) ([]finance.FinanceDocument, error) {
	items := make([]finance.FinanceDocument, 0)
	for rows.Next() {
		var item finance.FinanceDocument
		var payloadText string
		if err := rows.Scan(
			&item.ID, uuidPtrScanner(&item.TaxiParkID), uuidPtrScanner(&item.DriverID), uuidPtrScanner(&item.OrderID),
			uuidPtrScanner(&item.PayoutID), uuidPtrScanner(&item.InvoiceID), &item.Type, &item.Number, &item.Status,
			&item.FileURL, &payloadText, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan finance document: %w", err)
		}
		item.Payload = []byte(payloadText)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate finance documents: %w", err)
	}
	return items, nil
}

func nullableTime(target **time.Time) any {
	return &pgTimeScanner{target: target}
}

type pgTimeScanner struct {
	target **time.Time
}

func (scanner *pgTimeScanner) Scan(src any) error {
	switch value := src.(type) {
	case time.Time:
		copyValue := value
		*scanner.target = &copyValue
		return nil
	case nil:
		*scanner.target = nil
		return nil
	default:
		return fmt.Errorf("unsupported time source %T", src)
	}
}

func uuidPtrScanner(target **uuid.UUID) any {
	return &pgUUIDScanner{target: target}
}

type pgUUIDScanner struct {
	target **uuid.UUID
}

func (scanner *pgUUIDScanner) Scan(src any) error {
	switch value := src.(type) {
	case [16]byte:
		parsed := uuid.UUID(value)
		*scanner.target = &parsed
		return nil
	case []byte:
		parsed, err := uuid.FromBytes(value)
		if err != nil {
			return err
		}
		*scanner.target = &parsed
		return nil
	case string:
		parsed, err := uuid.Parse(value)
		if err != nil {
			return err
		}
		*scanner.target = &parsed
		return nil
	case nil:
		*scanner.target = nil
		return nil
	default:
		return fmt.Errorf("unsupported uuid source %T", src)
	}
}
