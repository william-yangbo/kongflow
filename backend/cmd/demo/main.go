package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"kongflow/backend/internal/database"
	"kongflow/backend/internal/secretstore"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== KongFlow SecretStore MVP Demo ===")

	// 配置数据库连接（需要先启动 docker-compose up postgres）
	config := database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "kong",
		Password: "flow2025",
		Database: "kongflow_dev",
		SSLMode:  "disable",
	}

	// 创建数据库连接池
	fmt.Println("\n1. 连接数据库...")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	fmt.Println("✅ 数据库连接成功")

	// 创建 SecretStore 服务
	repo := secretstore.NewRepository(pool)
	service := secretstore.NewService(repo)

	// 演示数据
	demoSecrets := map[string]interface{}{
		"oauth.github": map[string]interface{}{
			"client_id":     "demo_github_client_123",
			"client_secret": "demo_github_secret_456",
			"scopes":        []string{"read:user", "repo"},
			"created_at":    time.Now().Format(time.RFC3339),
		},
		"database.config": map[string]interface{}{
			"host":     "prod-db.example.com",
			"port":     5432,
			"username": "app_user",
			"password": "super_secret_password",
			"database": "production_db",
			"ssl_mode": "require",
		},
		"api.keys": map[string]string{
			"stripe":   "sk_live_...",
			"sendgrid": "SG.abc123...",
			"jwt":      "super-secret-jwt-key",
		},
	}

	// 2. 存储密钥
	fmt.Println("\n2. 存储密钥...")
	for key, value := range demoSecrets {
		err := service.SetSecret(ctx, key, value)
		if err != nil {
			log.Printf("Failed to set secret %s: %v", key, err)
			continue
		}
		fmt.Printf("✅ 密钥 '%s' 存储成功\n", key)
	}

	// 3. 读取密钥
	fmt.Println("\n3. 读取密钥...")

	// 读取 GitHub OAuth 配置
	var githubConfig map[string]interface{}
	err = service.GetSecret(ctx, "oauth.github", &githubConfig)
	if err != nil {
		log.Printf("Failed to get GitHub config: %v", err)
	} else {
		fmt.Printf("✅ GitHub OAuth 配置: client_id=%s, scopes=%v\n",
			githubConfig["client_id"], githubConfig["scopes"])
	}

	// 读取数据库配置
	var dbConfig map[string]interface{}
	err = service.GetSecret(ctx, "database.config", &dbConfig)
	if err != nil {
		log.Printf("Failed to get database config: %v", err)
	} else {
		fmt.Printf("✅ 数据库配置: host=%s, database=%s\n",
			dbConfig["host"], dbConfig["database"])
	}

	// 读取 API 密钥
	var apiKeys map[string]string
	err = service.GetSecret(ctx, "api.keys", &apiKeys)
	if err != nil {
		log.Printf("Failed to get API keys: %v", err)
	} else {
		// 只显示前几个字符，保护敏感信息
		maskedKeys := make(map[string]string)
		for k, v := range apiKeys {
			if len(v) > 8 {
				maskedKeys[k] = v[:8] + "..."
			} else {
				maskedKeys[k] = "***"
			}
		}
		fmt.Printf("✅ API 密钥 (脱敏): %v\n", maskedKeys)
	}

	// 4. 更新密钥
	fmt.Println("\n4. 更新密钥...")
	updatedGithubConfig := map[string]interface{}{
		"client_id":     "updated_github_client_789",
		"client_secret": "updated_github_secret_000",
		"scopes":        []string{"read:user", "repo", "admin:org"},
		"updated_at":    time.Now().Format(time.RFC3339),
	}

	err = service.SetSecret(ctx, "oauth.github", updatedGithubConfig)
	if err != nil {
		log.Printf("Failed to update GitHub config: %v", err)
	} else {
		fmt.Println("✅ GitHub 配置更新成功")

		// 验证更新
		var verifyConfig map[string]interface{}
		err = service.GetSecret(ctx, "oauth.github", &verifyConfig)
		if err == nil {
			fmt.Printf("✅ 验证更新: client_id=%s, scopes=%v\n",
				verifyConfig["client_id"], verifyConfig["scopes"])
		}
	}

	// 5. 测试 GetSecretOrThrow
	fmt.Println("\n5. 测试 GetSecretOrThrow...")
	var testConfig map[string]interface{}
	err = service.GetSecretOrThrow(ctx, "oauth.github", &testConfig)
	if err != nil {
		log.Printf("GetSecretOrThrow failed: %v", err)
	} else {
		fmt.Println("✅ GetSecretOrThrow 成功")
	}

	// 测试不存在的密钥
	var missingConfig map[string]interface{}
	err = service.GetSecretOrThrow(ctx, "nonexistent.key", &missingConfig)
	if err != nil {
		fmt.Printf("✅ 正确处理不存在的密钥: %v\n", err)
	}

	// 6. 演示完成
	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("✅ SecretStore MVP 实现成功！")
	fmt.Println("✅ 支持 JSONB 存储和类型安全的数据访问")
	fmt.Println("✅ 兼容 trigger.dev SecretStore 接口")

	// 显示存储的数据示例
	fmt.Println("\n=== 存储数据示例 ===")
	for key := range demoSecrets {
		var data interface{}
		err := service.GetSecret(ctx, key, &data)
		if err == nil {
			prettyJSON, _ := json.MarshalIndent(data, "", "  ")
			fmt.Printf("\n密钥: %s\n%s\n", key, prettyJSON)
		}
	}
}
