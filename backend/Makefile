.PHONY: build test test-unit test-integration test-e2e clean demo setup

# 项目变量
PROJECT_NAME := kongflow-backend
GO_MODULE := kongflow/backend

# 构建
build:
	go build -o bin/demo ./cmd/demo

# 运行演示
demo: build
	./bin/demo

# 测试
test: test-unit test-integration

# 单元测试（使用 Mock）
test-unit:
	@echo "运行单元测试..."
	go test ./internal/secretstore -run "TestService_" -v

# 集成测试（使用 TestContainers）
test-integration:
	@echo "运行集成测试..."
	go test ./internal/secretstore -run "TestRepositoryTestSuite" -v

# 端到端测试
test-e2e:
	@echo "运行端到端测试..."
	go test ./internal/secretstore -run "TestIntegrationTestSuite" -v

# 运行所有测试
test-all:
	@echo "运行所有测试..."
	go test ./internal/secretstore -v

# 测试覆盖率
test-coverage:
	@echo "生成测试覆盖率报告..."
	go test ./internal/secretstore -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 基准测试
test-bench:
	go test ./internal/secretstore -bench=. -benchmem

# 启动开发环境
dev-up:
	docker-compose up -d postgres
	@echo "等待 PostgreSQL 启动..."
	sleep 5
	@echo "运行数据库迁移..."
	@docker-compose exec postgres psql -U kong -d kongflow_dev -c "\i /docker-entrypoint-initdb.d/001_secret_store.sql" || echo "迁移可能已运行"

# 停止开发环境
dev-down:
	docker-compose down

# 清理
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker-compose down -v

# 项目设置
setup:
	go mod tidy
	go mod download

# 代码检查
lint:
	golangci-lint run

# 格式化代码
fmt:
	go fmt ./...

# 显示帮助
help:
	@echo "可用命令:"
	@echo "  build          - 构建项目"
	@echo "  demo           - 运行演示程序"
	@echo "  test           - 运行单元测试和集成测试"
	@echo "  test-unit      - 运行单元测试"
	@echo "  test-integration - 运行集成测试"
	@echo "  test-e2e       - 运行端到端测试"
	@echo "  test-all       - 运行所有测试"
	@echo "  test-coverage  - 生成测试覆盖率报告"
	@echo "  test-bench     - 运行基准测试"
	@echo "  dev-up         - 启动开发环境"
	@echo "  dev-down       - 停止开发环境"
	@echo "  setup          - 项目设置"
	@echo "  clean          - 清理构建文件"
	@echo "  fmt            - 格式化代码"
	@echo "  help           - 显示帮助"