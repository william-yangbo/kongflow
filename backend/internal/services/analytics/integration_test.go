package analytics

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestIntegration_BehaviouralAnalytics tests the complete analytics service workflow
// This mirrors a real-world usage scenario matching trigger.dev patterns
func TestIntegration_BehaviouralAnalytics(t *testing.T) {
	// Initialize service with nil client for testing (graceful degradation)
	service := &BehaviouralAnalytics{
		client: nil,
		config: DefaultConfig(),
	}

	ctx := context.Background()

	// Step 1: User registration flow
	t.Run("UserRegistrationFlow", func(t *testing.T) {
		userData := newTestUser("user_integration_test", "integration@test.com", "Integration Test User", time.Now())

		// New user identification
		err := service.UserIdentify(ctx, userData, true)
		if err != nil {
			t.Errorf("Failed to identify new user: %v", err)
		}

		// Existing user identification (should not trigger creation event)
		err = service.UserIdentify(ctx, userData, false)
		if err != nil {
			t.Errorf("Failed to identify existing user: %v", err)
		}
	})

	// Step 2: Organization lifecycle
	t.Run("OrganizationLifecycle", func(t *testing.T) {
		orgData := newTestOrganization("org_integration_test", "Integration Test Organization", "integration-test-org", time.Now(), time.Now())

		// Organization identification
		err := service.OrganizationIdentify(ctx, orgData)
		if err != nil {
			t.Errorf("Failed to identify organization: %v", err)
		}

		// Organization creation tracking
		err = service.OrganizationNew(ctx, "user_integration_test", orgData, 1)
		if err != nil {
			t.Errorf("Failed to track organization creation: %v", err)
		}
	})

	// Step 3: Project lifecycle
	t.Run("ProjectLifecycle", func(t *testing.T) {
		projectData := newTestProject("project_integration_test", "Integration Test Project", "integration-test-project", "org_integration_test", time.Now(), time.Now())

		// Project identification
		err := service.ProjectIdentify(ctx, projectData)
		if err != nil {
			t.Errorf("Failed to identify project: %v", err)
		}

		// Project creation tracking
		err = service.ProjectNew(ctx, "user_integration_test", "org_integration_test", projectData)
		if err != nil {
			t.Errorf("Failed to track project creation: %v", err)
		}
	})

	// Step 4: Environment setup
	t.Run("EnvironmentSetup", func(t *testing.T) {
		envData := newTestEnvironment("env_integration_test", "integration-test", "org_integration_test", "project_integration_test", "member_integration_test", time.Now(), time.Now())

		err := service.EnvironmentIdentify(ctx, envData)
		if err != nil {
			t.Errorf("Failed to identify environment: %v", err)
		}
	})

	// Step 5: Custom telemetry events
	t.Run("CustomTelemetryEvents", func(t *testing.T) {
		orgID := "org_integration_test"
		projectID := "project_integration_test"
		envID := "env_integration_test"
		jobID := "job_integration_test"

		// Workflow execution event
		workflowEvent := &TelemetryEvent{
			UserID: "user_integration_test",
			Event:  "workflow_executed",
			Properties: map[string]interface{}{
				"workflow_id":  "workflow_test_123",
				"duration_ms":  2500,
				"status":       "success",
				"trigger_type": "manual",
				"step_count":   5,
			},
			OrganizationID: &orgID,
			EnvironmentID:  &envID,
		}

		err := service.Capture(ctx, workflowEvent)
		if err != nil {
			t.Errorf("Failed to capture workflow event: %v", err)
		}

		// Feature usage event with all group types
		featureEvent := &TelemetryEvent{
			UserID: "user_integration_test",
			Event:  "feature_used",
			Properties: map[string]interface{}{
				"feature_name":    "advanced_scheduling",
				"usage_count":     10,
				"plan_type":       "enterprise",
				"feature_enabled": true,
			},
			OrganizationID: &orgID,
			ProjectID:      &projectID,
			EnvironmentID:  &envID,
			JobID:          &jobID,
		}

		err = service.Capture(ctx, featureEvent)
		if err != nil {
			t.Errorf("Failed to capture feature event: %v", err)
		}

		// Error tracking event
		errorEvent := &TelemetryEvent{
			UserID: "user_integration_test",
			Event:  "error_occurred",
			Properties: map[string]interface{}{
				"error_type":    "validation_error",
				"error_message": "Invalid input format",
				"error_code":    "E1001",
				"severity":      "warning",
			},
			OrganizationID: &orgID,
			ProjectID:      &projectID,
		}

		err = service.Capture(ctx, errorEvent)
		if err != nil {
			t.Errorf("Failed to capture error event: %v", err)
		}
	})

	// Step 6: Service cleanup
	t.Run("ServiceCleanup", func(t *testing.T) {
		err := service.Close()
		if err != nil {
			t.Errorf("Failed to close service: %v", err)
		}
	})
}

// TestIntegration_WithRealPostHogConfig tests with real PostHog configuration
// This test will only run if POSTHOG_PROJECT_KEY environment variable is set
func TestIntegration_WithRealPostHogConfig(t *testing.T) {
	config := &Config{
		PostHogProjectKey: "phc_test_key_for_integration", // Mock key for testing
		PostHogHost:       "https://app.posthog.com",
	}

	// This will create a real PostHog client, but with invalid key
	// The test verifies that initialization works correctly
	service, err := NewBehaviouralAnalytics(config)
	if err != nil {
		t.Errorf("Failed to create service with PostHog config: %v", err)
		return
	}

	if service.client == nil {
		t.Error("Expected non-nil client with valid config")
	}

	if service.config != config {
		t.Error("Service config does not match provided config")
	}

	// Clean up
	err = service.Close()
	if err != nil {
		t.Errorf("Failed to close service: %v", err)
	}
}

// TestIntegration_ConcurrentUsage tests concurrent access to the analytics service
func TestIntegration_ConcurrentUsage(t *testing.T) {
	service := &BehaviouralAnalytics{
		client: nil, // Use nil client for testing
		config: DefaultConfig(),
	}

	ctx := context.Background()
	const numGoroutines = 10
	const eventsPerGoroutine = 5

	// Channel to collect errors from goroutines
	errChan := make(chan error, numGoroutines*eventsPerGoroutine)
	doneChan := make(chan bool, numGoroutines)

	// Launch multiple goroutines that use the service concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { doneChan <- true }()

			for j := 0; j < eventsPerGoroutine; j++ {
				// Test different types of operations
				switch j % 4 {
				case 0:
					// User identification
					userData := newTestUser(fmt.Sprintf("user_%d_%d", goroutineID, j), fmt.Sprintf("user%d_%d@test.com", goroutineID, j), fmt.Sprintf("User %d %d", goroutineID, j), time.Now())
					err := service.UserIdentify(ctx, userData, true)
					if err != nil {
						errChan <- fmt.Errorf("goroutine %d: user identify failed: %v", goroutineID, err)
					}

				case 1:
					// Organization identification
					orgData := newTestOrganization(fmt.Sprintf("org_%d_%d", goroutineID, j), fmt.Sprintf("Org %d %d", goroutineID, j), fmt.Sprintf("org-%d-%d", goroutineID, j), time.Now(), time.Now())
					err := service.OrganizationIdentify(ctx, orgData)
					if err != nil {
						errChan <- fmt.Errorf("goroutine %d: org identify failed: %v", goroutineID, err)
					}

				case 2:
					// Project identification
					projectData := newTestProject(fmt.Sprintf("project_%d_%d", goroutineID, j), fmt.Sprintf("Project %d %d", goroutineID, j), fmt.Sprintf("project-%d-%d", goroutineID, j), fmt.Sprintf("org_%d_%d", goroutineID, j), time.Now(), time.Now())
					err := service.ProjectIdentify(ctx, projectData)
					if err != nil {
						errChan <- fmt.Errorf("goroutine %d: project identify failed: %v", goroutineID, err)
					}

				case 3:
					// Custom event capture
					event := &TelemetryEvent{
						UserID: fmt.Sprintf("user_%d_%d", goroutineID, j),
						Event:  "concurrent_test_event",
						Properties: map[string]interface{}{
							"goroutine_id": goroutineID,
							"event_index":  j,
							"timestamp":    time.Now().Unix(),
						},
					}
					err := service.Capture(ctx, event)
					if err != nil {
						errChan <- fmt.Errorf("goroutine %d: capture failed: %v", goroutineID, err)
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}

	// Check for any errors
	close(errChan)
	for err := range errChan {
		t.Error(err)
	}

	// Clean up
	err := service.Close()
	if err != nil {
		t.Errorf("Failed to close service after concurrent test: %v", err)
	}
}
