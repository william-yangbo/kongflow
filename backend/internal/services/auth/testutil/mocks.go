package testutil

import (
	"context"
	"net/http"
	"time"

	"kongflow/backend/internal/services/analytics"
	"kongflow/backend/internal/services/email"
	"kongflow/backend/internal/services/workerqueue"
	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/mock"
)

// MockQueries is a centralized mock for database queries
type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) FindUserByEmail(ctx context.Context, email string) (shared.Users, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(shared.Users), args.Error(1)
}

func (m *MockQueries) CreateUser(ctx context.Context, arg shared.CreateUserParams) (shared.Users, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(shared.Users), args.Error(1)
}

func (m *MockQueries) GetUser(ctx context.Context, id pgtype.UUID) (shared.Users, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(shared.Users), args.Error(1)
}

// Stub implementations for other required interface methods
func (m *MockQueries) CreateOrganization(ctx context.Context, arg shared.CreateOrganizationParams) (shared.Organizations, error) {
	return shared.Organizations{}, nil
}
func (m *MockQueries) CreateProject(ctx context.Context, arg shared.CreateProjectParams) (shared.Projects, error) {
	return shared.Projects{}, nil
}
func (m *MockQueries) CreateRuntimeEnvironment(ctx context.Context, arg shared.CreateRuntimeEnvironmentParams) (shared.RuntimeEnvironments, error) {
	return shared.RuntimeEnvironments{}, nil
}

// External Accounts stub implementations - 新增的外部账户相关方法
func (m *MockQueries) CreateExternalAccount(ctx context.Context, arg shared.CreateExternalAccountParams) (shared.ExternalAccounts, error) {
	return shared.ExternalAccounts{}, nil
}
func (m *MockQueries) FindExternalAccountByEnvAndIdentifier(ctx context.Context, arg shared.FindExternalAccountByEnvAndIdentifierParams) (shared.ExternalAccounts, error) {
	return shared.ExternalAccounts{}, nil
}
func (m *MockQueries) GetExternalAccountByID(ctx context.Context, id pgtype.UUID) (shared.ExternalAccounts, error) {
	return shared.ExternalAccounts{}, nil
}
func (m *MockQueries) ListExternalAccountsByEnvironment(ctx context.Context, arg shared.ListExternalAccountsByEnvironmentParams) ([]shared.ExternalAccounts, error) {
	return []shared.ExternalAccounts{}, nil
}
func (m *MockQueries) UpdateExternalAccountMetadata(ctx context.Context, arg shared.UpdateExternalAccountMetadataParams) error {
	return nil
}

func (m *MockQueries) FindOrganizationBySlug(ctx context.Context, slug string) (shared.Organizations, error) {
	return shared.Organizations{}, nil
}
func (m *MockQueries) FindProjectBySlug(ctx context.Context, arg shared.FindProjectBySlugParams) (shared.Projects, error) {
	return shared.Projects{}, nil
}
func (m *MockQueries) FindRuntimeEnvironmentByAPIKey(ctx context.Context, apiKey string) (shared.RuntimeEnvironments, error) {
	return shared.RuntimeEnvironments{}, nil
}
func (m *MockQueries) FindRuntimeEnvironmentByPublicAPIKey(ctx context.Context, apiKey string) (shared.RuntimeEnvironments, error) {
	return shared.RuntimeEnvironments{}, nil
}
func (m *MockQueries) GetEnvironmentWithProjectAndOrg(ctx context.Context, id pgtype.UUID) (shared.GetEnvironmentWithProjectAndOrgRow, error) {
	return shared.GetEnvironmentWithProjectAndOrgRow{}, nil
}
func (m *MockQueries) GetOrganization(ctx context.Context, id pgtype.UUID) (shared.Organizations, error) {
	return shared.Organizations{}, nil
}
func (m *MockQueries) GetProject(ctx context.Context, id pgtype.UUID) (shared.Projects, error) {
	return shared.Projects{}, nil
}
func (m *MockQueries) GetRuntimeEnvironment(ctx context.Context, id pgtype.UUID) (shared.RuntimeEnvironments, error) {
	return shared.RuntimeEnvironments{}, nil
}
func (m *MockQueries) ListOrganizations(ctx context.Context) ([]shared.Organizations, error) {
	return []shared.Organizations{}, nil
}
func (m *MockQueries) ListProjectsByOrganization(ctx context.Context, organizationID pgtype.UUID) ([]shared.Projects, error) {
	return []shared.Projects{}, nil
}
func (m *MockQueries) ListRuntimeEnvironmentsByProject(ctx context.Context, projectID pgtype.UUID) ([]shared.RuntimeEnvironments, error) {
	return []shared.RuntimeEnvironments{}, nil
}
func (m *MockQueries) ListUsers(ctx context.Context, arg shared.ListUsersParams) ([]shared.Users, error) {
	return []shared.Users{}, nil
}
func (m *MockQueries) UpdateOrganization(ctx context.Context, arg shared.UpdateOrganizationParams) (shared.Organizations, error) {
	return shared.Organizations{}, nil
}
func (m *MockQueries) UpdateProject(ctx context.Context, arg shared.UpdateProjectParams) (shared.Projects, error) {
	return shared.Projects{}, nil
}
func (m *MockQueries) UpdateRuntimeEnvironment(ctx context.Context, arg shared.UpdateRuntimeEnvironmentParams) (shared.RuntimeEnvironments, error) {
	return shared.RuntimeEnvironments{}, nil
}
func (m *MockQueries) UpdateUser(ctx context.Context, arg shared.UpdateUserParams) (shared.Users, error) {
	return shared.Users{}, nil
}

// MockEmailService is a mock for email functionality
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendMagicLinkEmail(ctx context.Context, opts email.SendMagicLinkOptions) error {
	args := m.Called(ctx, opts)
	return args.Error(0)
}

func (m *MockEmailService) ScheduleWelcomeEmail(ctx context.Context, user email.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockEmailService) ScheduleEmail(ctx context.Context, data email.DeliverEmail, delay *time.Duration) error {
	args := m.Called(ctx, data, delay)
	return args.Error(0)
}

func (m *MockEmailService) SendEmail(ctx context.Context, data email.DeliverEmail) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockEmailService) SetWorkerQueue(workerQueue *workerqueue.Client) {
	m.Called(workerQueue)
}

// MockAnalyticsService is a mock for analytics functionality
type MockAnalyticsService struct {
	mock.Mock
}

func (m *MockAnalyticsService) UserIdentify(ctx context.Context, user *shared.Users, isNewUser bool) error {
	args := m.Called(ctx, user, isNewUser)
	return args.Error(0)
}

func (m *MockAnalyticsService) OrganizationIdentify(ctx context.Context, org *shared.Organizations) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockAnalyticsService) OrganizationNew(ctx context.Context, userID string, org *shared.Organizations, organizationCount int) error {
	args := m.Called(ctx, userID, org, organizationCount)
	return args.Error(0)
}

func (m *MockAnalyticsService) ProjectIdentify(ctx context.Context, project *shared.Projects) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockAnalyticsService) ProjectNew(ctx context.Context, userID, organizationID string, project *shared.Projects) error {
	args := m.Called(ctx, userID, organizationID, project)
	return args.Error(0)
}

func (m *MockAnalyticsService) EnvironmentIdentify(ctx context.Context, env *shared.RuntimeEnvironments) error {
	args := m.Called(ctx, env)
	return args.Error(0)
}

func (m *MockAnalyticsService) Capture(ctx context.Context, event *analytics.TelemetryEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockAnalyticsService) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockImpersonationService is a mock for impersonation functionality
type MockImpersonationService struct {
	mock.Mock
}

func (m *MockImpersonationService) GetImpersonation(req *http.Request) (string, error) {
	args := m.Called(req)
	return args.String(0), args.Error(1)
}

func (m *MockImpersonationService) SetImpersonation(w http.ResponseWriter, req *http.Request, userID string) error {
	args := m.Called(w, req, userID)
	return args.Error(0)
}

func (m *MockImpersonationService) ClearImpersonation(w http.ResponseWriter, req *http.Request) error {
	args := m.Called(w, req)
	return args.Error(0)
}

// MockWorkerQueue is a mock for worker queue functionality
type MockWorkerQueue struct {
	mock.Mock
}

func (m *MockWorkerQueue) EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	args := m.Called(ctx, identifier, payload, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}

// MockSessionStorage is a mock for session storage functionality
type MockSessionStorage struct {
	mock.Mock
	sessions map[string]map[string]interface{}
}

func NewMockSessionStorage() *MockSessionStorage {
	return &MockSessionStorage{
		sessions: make(map[string]map[string]interface{}),
	}
}

func (m *MockSessionStorage) GetSession(ctx context.Context, req *http.Request) (map[string]interface{}, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return make(map[string]interface{}), args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockSessionStorage) CommitSession(ctx context.Context, w http.ResponseWriter, req *http.Request, session map[string]interface{}) error {
	args := m.Called(ctx, w, req, session)
	return args.Error(0)
}

func (m *MockSessionStorage) DestroySession(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	args := m.Called(ctx, w, req)
	return args.Error(0)
}

// MockPostAuthProcessor is a mock for post-authentication processing
type MockPostAuthProcessor struct {
	mock.Mock
}

func (m *MockPostAuthProcessor) PostAuthentication(ctx context.Context, user *shared.Users, isNewUser bool) error {
	args := m.Called(ctx, user, isNewUser)
	return args.Error(0)
}
