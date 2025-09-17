package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// SetupTestDB 创建测试数据库，运行所有迁移文件
func SetupTestDB(t *testing.T) *TestDB {
	return setupTestDBInternal(t, "")
}

// SetupTestDBWithMigrations 创建测试数据库，只运行包含指定模式的迁移文件
func SetupTestDBWithMigrations(t *testing.T, migrationPattern string) *TestDB {
	return setupTestDBInternal(t, migrationPattern)
}

func setupTestDBInternal(t *testing.T, migrationPattern string) *TestDB {
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

	// 运行迁移文件（与生产环境完全一致）
	if migrationPattern == "" {
		err = runAllMigrations(ctx, pool)
	} else {
		err = runFilteredMigrations(ctx, pool, migrationPattern)
	}
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

// runAllMigrations 运行所有迁移文件（与生产环境一致）
func runAllMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := getMigrationsDir()

	// 读取迁移文件目录
	migrationFiles, err := getMigrationFiles(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// 按文件名排序执行
	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {
		if err := executeMigrationFile(ctx, pool, filepath.Join(migrationsDir, file)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}

// runFilteredMigrations 运行匹配模式的迁移文件
func runFilteredMigrations(ctx context.Context, pool *pgxpool.Pool, pattern string) error {
	migrationsDir := getMigrationsDir()

	// 读取迁移文件目录
	allFiles, err := getMigrationFiles(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// 过滤匹配的文件
	var migrationFiles []string
	for _, file := range allFiles {
		if strings.Contains(file, pattern) {
			migrationFiles = append(migrationFiles, file)
		}
	}

	// 按文件名排序执行
	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {
		if err := executeMigrationFile(ctx, pool, filepath.Join(migrationsDir, file)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}

// getMigrationsDir 获取迁移文件目录的绝对路径
func getMigrationsDir() string {
	// 从当前工作目录向上查找迁移目录
	wd, _ := os.Getwd()

	// 尝试几个可能的路径
	possiblePaths := []string{
		filepath.Join(wd, "db", "migrations"),
		filepath.Join(wd, "..", "db", "migrations"),
		filepath.Join(wd, "..", "..", "db", "migrations"),
		filepath.Join(wd, "..", "..", "..", "db", "migrations"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// 默认返回相对路径
	return "../../db/migrations"
}

// getMigrationFiles 获取迁移目录下的所有.sql文件
func getMigrationFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// executeMigrationFile 执行单个迁移文件
func executeMigrationFile(ctx context.Context, pool *pgxpool.Pool, filePath string) error {
	// 读取SQL文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", filePath, err)
	}

	// 执行SQL
	_, err = pool.Exec(ctx, string(content))
	if err != nil {
		return fmt.Errorf("failed to execute SQL from %s: %w", filePath, err)
	}

	return nil
}
