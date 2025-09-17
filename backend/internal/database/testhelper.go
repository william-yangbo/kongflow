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
		CREATE TABLE IF NOT EXISTS "SecretStore" (
		    "key" TEXT NOT NULL,
		    "value" JSONB NOT NULL,
		    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
		    "updatedAt" TIMESTAMP(3) NOT NULL
		);

		-- CreateIndex
		CREATE UNIQUE INDEX IF NOT EXISTS "SecretStore_key_key" ON "SecretStore"("key");
	`
	_, err := pool.Exec(ctx, migration)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
