package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestHelper_MigrationReading(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	t.Run("测试所有迁移文件加载", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Cleanup(t)

		// 验证SecretStore表是否创建成功
		var exists bool
		err := db.Pool.QueryRow(context.Background(),
			`SELECT EXISTS (
				SELECT 1 FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'SecretStore'
			)`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "SecretStore表应该存在")

		// 验证索引是否创建成功
		var indexExists bool
		err = db.Pool.QueryRow(context.Background(),
			`SELECT EXISTS (
				SELECT 1 FROM pg_indexes 
				WHERE schemaname = 'public' 
				AND tablename = 'SecretStore' 
				AND indexname = 'SecretStore_key_key'
			)`).Scan(&indexExists)
		require.NoError(t, err)
		assert.True(t, indexExists, "SecretStore唯一索引应该存在")
	})

	t.Run("测试选择性迁移", func(t *testing.T) {
		db := SetupTestDBWithMigrations(t, "secret")
		defer db.Cleanup(t)

		// 验证SecretStore表是否创建成功
		var exists bool
		err := db.Pool.QueryRow(context.Background(),
			`SELECT EXISTS (
				SELECT 1 FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'SecretStore'
			)`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "SecretStore表应该存在")
	})

	t.Run("测试迁移目录查找", func(t *testing.T) {
		migrationsDir := getMigrationsDir()
		assert.NotEmpty(t, migrationsDir)

		files, err := getMigrationFiles(migrationsDir)
		require.NoError(t, err)
		assert.NotEmpty(t, files, "应该找到迁移文件")

		// 验证至少有secret_store迁移文件
		hasSecretStoreMigration := false
		for _, file := range files {
			if file == "001_secret_store.sql" {
				hasSecretStoreMigration = true
				break
			}
		}
		assert.True(t, hasSecretStoreMigration, "应该找到001_secret_store.sql文件")
	})
}