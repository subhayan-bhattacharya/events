package testdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/subhayan-bhattacharya/events/api/internal/testdb"
)

func TestTransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	database, err := testdb.Start(ctx)
	if err != nil {
		t.Fatalf("start test database: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := database.Close(cleanupCtx); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	if _, err := database.Pool.Exec(ctx, `CREATE TABLE rollback_probe (id integer PRIMARY KEY)`); err != nil {
		t.Fatalf("create probe table: %v", err)
	}

	tx, err := database.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO rollback_probe (id) VALUES (1)`); err != nil {
		t.Fatalf("insert probe row: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("roll back transaction: %v", err)
	}

	var count int
	if err := database.Pool.QueryRow(ctx, `SELECT count(*) FROM rollback_probe`).Scan(&count); err != nil {
		t.Fatalf("count probe rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove the row; got %d rows", count)
	}
}
