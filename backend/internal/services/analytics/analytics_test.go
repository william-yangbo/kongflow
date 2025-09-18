package analytics

import (
	"context"
	"testing"
	"time"

	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
)

// Helper functions to create test data with proper pgtype conversions

func newTestUser(id, email, name string, createdAt time.Time) *User {
	userUUID := pgtype.UUID{}
	userUUID.Scan(id) // In a real scenario, this would be a proper UUID

	return &shared.Users{
		ID:        userUUID,
		Email:     email,
		Name:      pgtype.Text{String: name, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
	}
}

func newTestOrganization(id, title, slug string, createdAt, updatedAt time.Time) *Organization {
	orgUUID := pgtype.UUID{}
	orgUUID.Scan(id) // In a real scenario, this would be a proper UUID

	return &shared.Organizations{
		ID:        orgUUID,
		Title:     title,
		Slug:      slug,
		CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
	}
}

func newTestProject(id, name, slug, orgID string, createdAt, updatedAt time.Time) *Project {
	projectUUID := pgtype.UUID{}
	projectUUID.Scan(id) // In a real scenario, this would be a proper UUID

	orgUUID := pgtype.UUID{}
	orgUUID.Scan(orgID)

	return &shared.Projects{
		ID:             projectUUID,
		Name:           name,
		Slug:           slug,
		OrganizationID: orgUUID,
		CreatedAt:      pgtype.Timestamptz{Time: createdAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: updatedAt, Valid: true},
	}
}

func newTestEnvironment(id, slug, orgID, projectID, memberID string, createdAt, updatedAt time.Time) *Environment {
	envUUID := pgtype.UUID{}
	envUUID.Scan(id) // In a real scenario, this would be a proper UUID

	orgUUID := pgtype.UUID{}
	orgUUID.Scan(orgID)

	projUUID := pgtype.UUID{}
	projUUID.Scan(projectID)

	memberUUID := pgtype.UUID{}
	memberUUID.Scan(memberID)

	return &shared.RuntimeEnvironments{
		ID:             envUUID,
		Slug:           slug,
		ApiKey:         "test-api-key",
		Type:           "DEVELOPMENT",
		OrganizationID: orgUUID,
		ProjectID:      projUUID,
		OrgMemberID:    memberUUID,
		CreatedAt:      pgtype.Timestamptz{Time: createdAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: updatedAt, Valid: true},
	}
}

func TestNewBehaviouralAnalytics(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		expectNil   bool
	}{
		{
			name:        "nil config uses default",
			config:      nil,
			expectError: false,
			expectNil:   false,
		},
		{
			name: "empty API key creates service with nil client",
			config: &Config{
				PostHogProjectKey: "",
				PostHogHost:       "https://app.posthog.com",
			},
			expectError: false,
			expectNil:   false,
		},
		{
			name: "valid config creates service",
			config: &Config{
				PostHogProjectKey: "phc_test_key",
				PostHogHost:       "https://app.posthog.com",
			},
			expectError: false,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewBehaviouralAnalytics(tt.config)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectNil && service != nil {
				t.Errorf("expected nil service but got %v", service)
			}
			if !tt.expectNil && service == nil {
				t.Errorf("expected non-nil service but got nil")
			}

			// Test graceful degradation - service should handle nil client
			if service != nil && service.client == nil && tt.config != nil && tt.config.PostHogProjectKey == "" {
				// This is expected behavior for empty API key
				t.Logf("Service correctly created with nil client for empty API key")
			}
		})
	}
}

func TestBehaviouralAnalytics_UserIdentify(t *testing.T) {
	// Test with nil client (graceful degradation)
	service := &BehaviouralAnalytics{
		client: nil,
		config: DefaultConfig(),
	}

	ctx := context.Background()
	userData := newTestUser("user123", "test@example.com", "Test User", time.Now())

	// Should not error with nil client
	err := service.UserIdentify(ctx, userData, true)
	if err != nil {
		t.Errorf("UserIdentify with nil client should not error: %v", err)
	}

	// Test with new user flag
	err = service.UserIdentify(ctx, userData, false)
	if err != nil {
		t.Errorf("UserIdentify with existing user should not error: %v", err)
	}
}

func TestBehaviouralAnalytics_OrganizationIdentify(t *testing.T) {
	service := &BehaviouralAnalytics{
		client: nil,
		config: DefaultConfig(),
	}

	ctx := context.Background()
	orgData := newTestOrganization("org123", "Test Organization", "test-org", time.Now(), time.Now())

	err := service.OrganizationIdentify(ctx, orgData)
	if err != nil {
		t.Errorf("OrganizationIdentify with nil client should not error: %v", err)
	}
}

func TestBehaviouralAnalytics_ProjectIdentify(t *testing.T) {
	service := &BehaviouralAnalytics{
		client: nil,
		config: DefaultConfig(),
	}

	ctx := context.Background()
	projectData := newTestProject("project123", "Test Project", "test-project", "org123", time.Now(), time.Now())

	err := service.ProjectIdentify(ctx, projectData)
	if err != nil {
		t.Errorf("ProjectIdentify with nil client should not error: %v", err)
	}
}

func TestBehaviouralAnalytics_EnvironmentIdentify(t *testing.T) {
	service := &BehaviouralAnalytics{
		client: nil,
		config: DefaultConfig(),
	}

	ctx := context.Background()
	envData := newTestEnvironment("env123", "production", "org123", "project123", "member123", time.Now(), time.Now())

	err := service.EnvironmentIdentify(ctx, envData)
	if err != nil {
		t.Errorf("EnvironmentIdentify with nil client should not error: %v", err)
	}
}

func TestBehaviouralAnalytics_Capture(t *testing.T) {
	service := &BehaviouralAnalytics{
		client: nil,
		config: DefaultConfig(),
	}

	ctx := context.Background()
	orgID := "org123"
	envID := "env123"

	event := &TelemetryEvent{
		UserID: "user123",
		Event:  "test_event",
		Properties: map[string]interface{}{
			"property1": "value1",
			"property2": 42,
		},
		OrganizationID: &orgID,
		EnvironmentID:  &envID,
	}

	err := service.Capture(ctx, event)
	if err != nil {
		t.Errorf("Capture with nil client should not error: %v", err)
	}
}

func TestBehaviouralAnalytics_Close(t *testing.T) {
	service := &BehaviouralAnalytics{
		client: nil,
		config: DefaultConfig(),
	}

	err := service.Close()
	if err != nil {
		t.Errorf("Close with nil client should not error: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Error("DefaultConfig should not return nil")
		return
	}
	if config.PostHogHost == "" {
		t.Error("DefaultConfig should have non-empty PostHogHost")
	}
}
