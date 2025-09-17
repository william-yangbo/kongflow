package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
	Container testcontainers.Container
	Pool      *pgxpool.Pool
	Config    Config
}

func SetupTestDB(t *testing.T) *TestDB {
	ctx := context.Background()

	// 启动 PostgreSQL TestContainer
	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	// 获取连接信息
	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	config := Config{
		Host:     host,
		Port:     port.Int(),
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	// 创建连接池
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)

	// 运行迁移
	err = runMigrations(ctx, pool)
	require.NoError(t, err)

	return &TestDB{
		Container: container,
		Pool:      pool,
		Config:    config,
	}
}

func (db *TestDB) Cleanup(t *testing.T) {
	ctx := context.Background()
	if db.Pool != nil {
		db.Pool.Close()
	}
	if db.Container != nil {
		require.NoError(t, db.Container.Terminate(ctx))
	}
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migration := `
		CREATE TABLE IF NOT EXISTS secret_store (
		    key TEXT PRIMARY KEY,
		    value JSONB NOT NULL,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create index for faster lookups
		CREATE INDEX IF NOT EXISTS idx_secret_store_created_at ON secret_store(created_at);
		CREATE INDEX IF NOT EXISTS idx_secret_store_updated_at ON secret_store(updated_at);

		-- Create function to update updated_at automatically
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
		    NEW.updated_at = NOW();
		    RETURN NEW;
		END;
		$$ language 'plpgsql';

		-- Create trigger to auto-update updated_at
		DROP TRIGGER IF EXISTS update_secret_store_updated_at ON secret_store;
		CREATE TRIGGER update_secret_store_updated_at
		    BEFORE UPDATE ON secret_store
		    FOR EACH ROW
		    EXECUTE FUNCTION update_updated_at_column();
	`
	_, err := pool.Exec(ctx, migration)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
