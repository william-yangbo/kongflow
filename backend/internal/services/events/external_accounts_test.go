package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"kongflow/backend/internal/database"
	"kongflow/backend/internal/shared"
)

// ExternalAccountsIntegrationTestSuite 测试外部账户集成功能
type ExternalAccountsIntegrationTestSuite struct {
	suite.Suite
	db            *database.TestDB
	sharedQueries *shared.Queries
	eventQueries  *Queries
	testOrgID     uuid.UUID
	testProjectID uuid.UUID
	testEnvID     uuid.UUID
}

func (suite *ExternalAccountsIntegrationTestSuite) SetupSuite() {
	suite.db = database.SetupTestDB(suite.T())
	suite.sharedQueries = shared.New(suite.db.Pool)
	suite.eventQueries = New(suite.db.Pool)

	// 初始化测试IDs
	suite.testOrgID = uuid.New()
	suite.testProjectID = uuid.New()
	suite.testEnvID = uuid.New()
}

func (suite *ExternalAccountsIntegrationTestSuite) TearDownSuite() {
	suite.db.Cleanup(suite.T())
}

func (suite *ExternalAccountsIntegrationTestSuite) SetupTest() {
	ctx := context.Background()

	// 清理所有测试数据
	_, err := suite.db.Pool.Exec(ctx, `DELETE FROM event_records`)
	require.NoError(suite.T(), err)
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM external_accounts`)
	require.NoError(suite.T(), err)
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM runtime_environments`)
	require.NoError(suite.T(), err)
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM projects`)
	require.NoError(suite.T(), err)
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM organizations`)
	require.NoError(suite.T(), err)

	// 创建测试组织
	_, err = suite.db.Pool.Exec(ctx, `
		INSERT INTO organizations (id, title, slug) 
		VALUES ($1, $2, $3)`,
		suite.testOrgID, "Test Organization", "test-org")
	require.NoError(suite.T(), err)

	// 创建测试项目
	_, err = suite.db.Pool.Exec(ctx,
		`INSERT INTO projects (id, name, slug, organization_id, created_at, updated_at) 
		 VALUES ($1, 'Test Project', 'test-project', $2, NOW(), NOW())`,
		suite.testProjectID, suite.testOrgID)
	require.NoError(suite.T(), err)

	// 创建测试环境
	_, err = suite.db.Pool.Exec(ctx,
		`INSERT INTO runtime_environments (id, slug, api_key, type, organization_id, project_id, created_at, updated_at) 
		 VALUES ($1, 'test-env', 'test-api-key', 'DEVELOPMENT', $2, $3, NOW(), NOW())`,
		suite.testEnvID, suite.testOrgID, suite.testProjectID)
	require.NoError(suite.T(), err)
}

// TestExternalAccountQueryIntegration 测试外部账户查询集成
func (suite *ExternalAccountsIntegrationTestSuite) TestExternalAccountQueryIntegration() {
	ctx := context.Background()

	// 创建外部账户
	externalAccountID := uuid.New()
	_, err := suite.db.Pool.Exec(ctx, `
		INSERT INTO external_accounts (id, identifier, metadata, environment_id, organization_id) 
		VALUES ($1, $2, $3, $4, $5)`,
		externalAccountID, "user-123", `{"name": "Test User", "email": "test@example.com"}`,
		suite.testEnvID, suite.testOrgID)
	require.NoError(suite.T(), err)

	// 测试通过 shared queries 查找外部账户
	params := shared.FindExternalAccountByEnvAndIdentifierParams{
		EnvironmentID: pgtype.UUID{Bytes: suite.testEnvID, Valid: true},
		Identifier:    "user-123",
	}

	account, err := suite.sharedQueries.FindExternalAccountByEnvAndIdentifier(ctx, params)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), externalAccountID[:], account.ID.Bytes[:])
	assert.Equal(suite.T(), "user-123", account.Identifier)
	assert.Equal(suite.T(), suite.testEnvID[:], account.EnvironmentID.Bytes[:])
}

// TestExternalAccountWithEventRecord 测试外部账户与事件记录的关联
func (suite *ExternalAccountsIntegrationTestSuite) TestExternalAccountWithEventRecord() {
	ctx := context.Background()

	// 创建外部账户
	externalAccountID := uuid.New()
	_, err := suite.db.Pool.Exec(ctx, `
		INSERT INTO external_accounts (id, identifier, metadata, environment_id, organization_id) 
		VALUES ($1, $2, $3, $4, $5)`,
		externalAccountID, "premium-user-456", `{"tier": "premium", "region": "us-east"}`,
		suite.testEnvID, suite.testOrgID)
	require.NoError(suite.T(), err)

	// 创建关联外部账户的事件记录
	eventID := "event-" + uuid.New().String()
	createParams := CreateEventRecordParams{
		EventID:           eventID,
		Name:              "billing.invoice_ready",
		Timestamp:         pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Payload:           []byte(`{"invoiceId": "inv-789", "amount": 99.99}`),
		Context:           []byte(`{"source": "billing_system"}`),
		Source:            "webhook",
		OrganizationID:    pgtype.UUID{Bytes: suite.testOrgID, Valid: true},
		EnvironmentID:     pgtype.UUID{Bytes: suite.testEnvID, Valid: true},
		ProjectID:         pgtype.UUID{Bytes: suite.testProjectID, Valid: true},
		ExternalAccountID: pgtype.UUID{Bytes: externalAccountID, Valid: true}, // 关联外部账户
		DeliverAt:         pgtype.Timestamptz{Time: time.Now().Add(5 * time.Minute), Valid: true},
		IsTest:            false,
	}

	eventRecord, err := suite.eventQueries.CreateEventRecord(ctx, createParams)
	require.NoError(suite.T(), err)

	// 验证事件记录正确关联外部账户
	assert.True(suite.T(), eventRecord.ExternalAccountID.Valid)
	assert.Equal(suite.T(), externalAccountID[:], eventRecord.ExternalAccountID.Bytes[:])
	assert.Equal(suite.T(), eventID, eventRecord.EventID)

	// 验证可以通过事件ID查询到记录
	getParams := GetEventRecordByEventIDParams{
		EventID:       eventID,
		EnvironmentID: pgtype.UUID{Bytes: suite.testEnvID, Valid: true},
	}
	retrieved, err := suite.eventQueries.GetEventRecordByEventID(ctx, getParams)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), externalAccountID[:], retrieved.ExternalAccountID.Bytes[:])
}

// TestExternalAccountNotFound 测试外部账户不存在的情况
func (suite *ExternalAccountsIntegrationTestSuite) TestExternalAccountNotFound() {
	ctx := context.Background()

	// 尝试查找不存在的外部账户
	params := shared.FindExternalAccountByEnvAndIdentifierParams{
		EnvironmentID: pgtype.UUID{Bytes: suite.testEnvID, Valid: true},
		Identifier:    "nonexistent-user",
	}

	_, err := suite.sharedQueries.FindExternalAccountByEnvAndIdentifier(ctx, params)
	assert.Error(suite.T(), err)
	// 应该是 "no rows in result set" 或类似的数据库错误
	assert.Contains(suite.T(), err.Error(), "no rows")
}

// TestEventRecordWithoutExternalAccount 测试没有外部账户的事件记录
func (suite *ExternalAccountsIntegrationTestSuite) TestEventRecordWithoutExternalAccount() {
	ctx := context.Background()

	// 创建没有外部账户的事件记录
	eventID := "event-" + uuid.New().String()
	createParams := CreateEventRecordParams{
		EventID:           eventID,
		Name:              "system.notification",
		Timestamp:         pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Payload:           []byte(`{"message": "System maintenance scheduled"}`),
		Context:           []byte(`{"source": "system"}`),
		Source:            "internal",
		OrganizationID:    pgtype.UUID{Bytes: suite.testOrgID, Valid: true},
		EnvironmentID:     pgtype.UUID{Bytes: suite.testEnvID, Valid: true},
		ProjectID:         pgtype.UUID{Bytes: suite.testProjectID, Valid: true},
		ExternalAccountID: pgtype.UUID{}, // 不关联外部账户 (NULL)
		DeliverAt:         pgtype.Timestamptz{Time: time.Now().Add(1 * time.Minute), Valid: true},
		IsTest:            false,
	}

	eventRecord, err := suite.eventQueries.CreateEventRecord(ctx, createParams)
	require.NoError(suite.T(), err)

	// 验证外部账户ID为空
	assert.False(suite.T(), eventRecord.ExternalAccountID.Valid)
	assert.Equal(suite.T(), eventID, eventRecord.EventID)
}

// TestExternalAccountDataModel 测试外部账户数据模型结构对齐 trigger.dev
func (suite *ExternalAccountsIntegrationTestSuite) TestExternalAccountDataModel() {
	ctx := context.Background()

	// 创建复杂的外部账户元数据，对齐 trigger.dev ExternalAccount 模型
	metadata := `{
		"id": "user-789",
		"name": "Advanced User", 
		"email": "advanced@example.com",
		"tier": "enterprise",
		"features": ["webhooks", "api_access", "priority_support"],
		"limits": {
			"requests_per_minute": 1000,
			"storage_gb": 100
		},
		"created_at": "2024-01-15T10:30:00Z"
	}`

	externalAccountID := uuid.New()
	_, err := suite.db.Pool.Exec(ctx, `
		INSERT INTO external_accounts (id, identifier, metadata, environment_id, organization_id) 
		VALUES ($1, $2, $3, $4, $5)`,
		externalAccountID, "enterprise-user-789", metadata,
		suite.testEnvID, suite.testOrgID)
	require.NoError(suite.T(), err)

	// 通过 shared queries 查询验证
	params := shared.FindExternalAccountByEnvAndIdentifierParams{
		EnvironmentID: pgtype.UUID{Bytes: suite.testEnvID, Valid: true},
		Identifier:    "enterprise-user-789",
	}

	account, err := suite.sharedQueries.FindExternalAccountByEnvAndIdentifier(ctx, params)
	require.NoError(suite.T(), err)

	// 验证数据模型字段
	assert.Equal(suite.T(), externalAccountID[:], account.ID.Bytes[:])
	assert.Equal(suite.T(), "enterprise-user-789", account.Identifier)
	assert.Equal(suite.T(), suite.testEnvID[:], account.EnvironmentID.Bytes[:])
	assert.Equal(suite.T(), suite.testOrgID[:], account.OrganizationID.Bytes[:])

	// JSON比较 - 解析后比较内容而非字符串格式
	var expectedMeta, actualMeta map[string]interface{}
	err = json.Unmarshal([]byte(metadata), &expectedMeta)
	require.NoError(suite.T(), err)
	err = json.Unmarshal(account.Metadata, &actualMeta)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedMeta, actualMeta)

	assert.True(suite.T(), account.CreatedAt.Valid)
	assert.True(suite.T(), account.UpdatedAt.Valid)
}

// TestRunner 运行外部账户集成测试套件
func TestExternalAccountsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExternalAccountsIntegrationTestSuite))
}
