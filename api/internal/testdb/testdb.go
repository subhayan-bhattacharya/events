// Package testdb provides PostgreSQL containers for integration tests.
package testdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	appPostgres "github.com/subhayan-bhattacharya/events/api/internal/postgres"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const image = "postgres:17-alpine"

// Database is an isolated PostgreSQL instance for an integration test suite.
type Database struct {
	Pool *pgxpool.Pool
	URL  string

	container *tcPostgres.PostgresContainer
}

// Start launches PostgreSQL and establishes a connection pool.
// The caller must call Close when the test suite finishes.
func Start(ctx context.Context) (*Database, error) {
	container, err := tcPostgres.Run(
		ctx,
		image,
		tcPostgres.WithDatabase("events_test"),
		tcPostgres.WithUsername("postgres"),
		tcPostgres.WithPassword("postgres"),
		tcPostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, terminateAfterError(ctx, container, "get postgres connection string", err)
	}

	pool, err := appPostgres.Connect(ctx, url)
	if err != nil {
		return nil, terminateAfterError(ctx, container, "connect to test postgres", err)
	}

	return &Database{
		Pool:      pool,
		URL:       url,
		container: container,
	}, nil
}

// Begin starts a transaction for one test. The caller must roll it back.
func (db *Database) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin test transaction: %w", err)
	}
	return tx, nil
}

// Close closes the pool and terminates the PostgreSQL container.
func (db *Database) Close(ctx context.Context) error {
	db.Pool.Close()
	if err := db.container.Terminate(ctx); err != nil {
		return fmt.Errorf("terminate postgres container: %w", err)
	}
	return nil
}

func terminateAfterError(
	ctx context.Context,
	container *tcPostgres.PostgresContainer,
	operation string,
	operationErr error,
) error {
	terminateErr := container.Terminate(ctx)
	if terminateErr != nil {
		terminateErr = fmt.Errorf("terminate postgres container: %w", terminateErr)
	}
	return errors.Join(fmt.Errorf("%s: %w", operation, operationErr), terminateErr)
}
