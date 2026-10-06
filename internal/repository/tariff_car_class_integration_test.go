package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTariffCarClassMigrationEnforcesNotNull(t *testing.T) {
	databaseURL := os.Getenv("TAXI_TARIFF_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TAXI_TARIFF_TEST_DATABASE_URL to a disposable database migrated through 000041")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var nullable string
	err = pool.QueryRow(context.Background(), `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'taxi_park_tariffs'
		  AND column_name = 'car_class_id'
	`).Scan(&nullable)
	if err != nil {
		t.Fatal(err)
	}
	if nullable != "NO" {
		t.Fatalf("taxi_park_tariffs.car_class_id is_nullable=%q, want NO", nullable)
	}

	var nullTariffCount int
	err = pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM taxi_park_tariffs
		WHERE car_class_id IS NULL
	`).Scan(&nullTariffCount)
	if err != nil {
		t.Fatal(err)
	}
	if nullTariffCount != 0 {
		t.Fatalf("found %d taxi park tariffs without car class", nullTariffCount)
	}

	transaction, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = transaction.Rollback(context.Background())
	}()

	commandTag, err := transaction.Exec(context.Background(), `
		UPDATE taxi_park_tariffs
		SET car_class_id = NULL
		WHERE id = (SELECT id FROM taxi_park_tariffs LIMIT 1)
	`)
	if commandTag.RowsAffected() == 0 && err == nil {
		t.Skip("database contains no taxi park tariff for the constraint check")
	}
	var postgresError *pgconn.PgError
	if err == nil {
		t.Fatal("setting tariff car_class_id to NULL succeeded")
	}
	if !errors.As(err, &postgresError) {
		t.Fatalf("setting tariff car_class_id to NULL returned %T, want PostgreSQL error", err)
	}
	if postgresError.Code != "23502" {
		t.Fatalf("setting tariff car_class_id to NULL returned SQLSTATE %s, want 23502", postgresError.Code)
	}
}
