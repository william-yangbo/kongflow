package analytics

import (
	"context"
	"testing"
	"time"
)

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
	userData := &UserData{
		ID:                   "user123",
		Email:                "test@example.com",
		Name:                 "Test User",
		AuthenticationMethod: "email",
		Admin:                false,
		CreatedAt:            time.Now(),
	}

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
	orgData := &OrganizationData{
		ID:        "org123",
		Title:     "Test Organization",
		Slug:      "test-org",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

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
	projectData := &ProjectData{
		ID:        "project123",
		Name:      "Test Project",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

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
	envData := &EnvironmentData{
		ID:             "env123",
		Slug:           "production",
		OrganizationID: "org123",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

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
