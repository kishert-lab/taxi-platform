package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func driverParkBalanceInTransaction(ctx context.Context, transaction pgx.Tx, driver uuid.UUID, park *uuid.UUID) (int64, error) {
	var balance int64
	if err := transaction.QueryRow(ctx, `SELECT available_balance_cents FROM driver_balances WHERE driver_id=$1 AND taxi_park_id IS NOT DISTINCT FROM $2::uuid`, driver, park).Scan(&balance); err != nil {
		return 0, fmt.Errorf("select driver park ledger balance: %w", err)
	}
	return balance, nil
}
