package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAssignDriverQueryCanBePrepared(t *testing.T) {
	databaseURL := os.Getenv("TAXI_DISPATCH_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TAXI_DISPATCH_TEST_DATABASE_URL to a disposable migrated database")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	connection, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()

	const statementName = "test_assign_driver_query"
	if _, err := connection.Conn().Prepare(context.Background(), statementName, assignDriverQuery); err != nil {
		t.Fatalf("prepare assign driver query: %v", err)
	}
	if err := connection.Conn().Deallocate(context.Background(), statementName); err != nil {
		t.Fatalf("deallocate assign driver query: %v", err)
	}
}
