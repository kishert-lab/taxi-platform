package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	taxiparkapp "github.com/kishert-lab/taxi-platform/internal/taxipark"
)

// GetOrderTariff resolves the same park/global tariff IDs accepted by order creation.
// The short read transaction finishes before any routing HTTP request.
func (repository *PostgresTaxiParkSettingsRepository) GetOrderTariff(ctx context.Context, actorID, tariffID uuid.UUID) (domain.TaxiParkTariff, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.TaxiParkTariff{}, fmt.Errorf("begin tariff read: %w", err)
	}
	defer rollbackTx(ctx, transaction)
	parkID, err := taxiParkIDByActor(ctx, transaction, actorID)
	if err != nil {
		return domain.TaxiParkTariff{}, err
	}
	tariff, err := scanTaxiParkTariff(transaction.QueryRow(ctx, `SELECT `+taxiParkTariffSelectColumns+` FROM taxi_park_tariffs t WHERE t.id=$1 AND t.taxi_park_id=$2 AND t.is_active=true`, tariffID, parkID))
	if errors.Is(err, pgx.ErrNoRows) {
		tariff = domain.TaxiParkTariff{ID: tariffID, PricingMode: domain.PricingModeDistance, IsActive: true}
		err = transaction.QueryRow(ctx, `SELECT (base_price*100)::bigint,(price_per_km*100)::bigint,(minimum_price*100)::bigint FROM tariffs WHERE id=$1 AND is_active=true AND deleted_at IS NULL`, tariffID).Scan(&tariff.BasePrice.Amount, &tariff.PricePerKM.Amount, &tariff.MinimumPrice.Amount)
		tariff.BasePrice.Currency = "RUB"
		tariff.PricePerKM.Currency = "RUB"
		tariff.MinimumPrice.Currency = "RUB"
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TaxiParkTariff{}, taxiparkapp.ErrOrderTariffNotFound
	}
	if err != nil {
		return domain.TaxiParkTariff{}, fmt.Errorf("read order tariff: %w", err)
	}
	if err = transaction.Commit(ctx); err != nil {
		return domain.TaxiParkTariff{}, fmt.Errorf("commit tariff read: %w", err)
	}
	return tariff, nil
}

func snapshotPriceCents(snapshot *domain.OrderPricingSnapshot) *int64 {
	if snapshot == nil || !snapshot.PriceAvailable || snapshot.EstimatedPrice == nil {
		return nil
	}
	return &snapshot.EstimatedPrice.Amount
}
