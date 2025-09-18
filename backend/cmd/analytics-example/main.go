package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"kongflow/backend/internal/services/analytics"
)

// Example demonstrating how to use the analytics service
// This aligns with trigger.dev's BehaviouralAnalytics usage patterns
func main() {
	// Initialize analytics service with configuration
	// In production, you would get the API key from environment variables
	config := &analytics.Config{
		PostHogProjectKey: os.Getenv("POSTHOG_PROJECT_KEY"), // Can be empty for testing
		PostHogHost:       "https://app.posthog.com",
	}

	service, err := analytics.NewBehaviouralAnalytics(config)
	if err != nil {
		log.Fatalf("Failed to create analytics service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()

	// Example 1: User registration flow
	fmt.Println("=== User Registration Flow ===")
	userData := &analytics.UserData{
		ID:                   "user_12345",
		Email:                "john.doe@example.com",
		Name:                 "John Doe",
		AuthenticationMethod: "email",
		Admin:                false,
		CreatedAt:            time.Now(),
	}

	// Track user identification (new user)
	err = service.UserIdentify(ctx, userData, true)
	if err != nil {
		log.Printf("Failed to identify user: %v", err)
	} else {
		fmt.Printf("✓ Identified new user: %s\n", userData.Name)
	}

	// Example 2: Organization creation
	fmt.Println("\n=== Organization Creation ===")
	orgData := &analytics.OrganizationData{
		ID:        "org_67890",
		Title:     "Acme Corporation",
		Slug:      "acme-corp",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Identify organization for grouping
	err = service.OrganizationIdentify(ctx, orgData)
	if err != nil {
		log.Printf("Failed to identify organization: %v", err)
	} else {
		fmt.Printf("✓ Identified organization: %s\n", orgData.Title)
	}

	// Track organization creation event
	err = service.OrganizationNew(ctx, userData.ID, orgData, 1)
	if err != nil {
		log.Printf("Failed to track organization creation: %v", err)
	} else {
		fmt.Printf("✓ Tracked organization creation for user: %s\n", userData.Name)
	}

	// Example 3: Project creation
	fmt.Println("\n=== Project Creation ===")
	projectData := &analytics.ProjectData{
		ID:        "project_abc123",
		Name:      "My Awesome Project",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Identify project for grouping
	err = service.ProjectIdentify(ctx, projectData)
	if err != nil {
		log.Printf("Failed to identify project: %v", err)
	} else {
		fmt.Printf("✓ Identified project: %s\n", projectData.Name)
	}

	// Track project creation event
	err = service.ProjectNew(ctx, userData.ID, orgData.ID, projectData)
	if err != nil {
		log.Printf("Failed to track project creation: %v", err)
	} else {
		fmt.Printf("✓ Tracked project creation: %s\n", projectData.Name)
	}

	// Example 4: Environment setup
	fmt.Println("\n=== Environment Setup ===")
	envData := &analytics.EnvironmentData{
		ID:             "env_production",
		Slug:           "production",
		OrganizationID: orgData.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Identify environment for grouping
	err = service.EnvironmentIdentify(ctx, envData)
	if err != nil {
		log.Printf("Failed to identify environment: %v", err)
	} else {
		fmt.Printf("✓ Identified environment: %s\n", envData.Slug)
	}

	// Example 5: Custom telemetry events
	fmt.Println("\n=== Custom Telemetry ===")

	// Track a custom workflow event
	workflowEvent := &analytics.TelemetryEvent{
		UserID: userData.ID,
		Event:  "workflow_executed",
		Properties: map[string]interface{}{
			"workflow_id":  "workflow_xyz789",
			"duration_ms":  1500,
			"status":       "success",
			"trigger_type": "manual",
		},
		OrganizationID: &orgData.ID,
		EnvironmentID:  &envData.ID,
	}

	err = service.Capture(ctx, workflowEvent)
	if err != nil {
		log.Printf("Failed to capture workflow event: %v", err)
	} else {
		fmt.Printf("✓ Tracked workflow execution event\n")
	}

	// Track a feature usage event
	featureEvent := &analytics.TelemetryEvent{
		UserID: userData.ID,
		Event:  "feature_used",
		Properties: map[string]interface{}{
			"feature_name": "advanced_scheduling",
			"usage_count":  5,
			"plan_type":    "pro",
		},
		OrganizationID: &orgData.ID,
	}

	err = service.Capture(ctx, featureEvent)
	if err != nil {
		log.Printf("Failed to capture feature event: %v", err)
	} else {
		fmt.Printf("✓ Tracked feature usage event\n")
	}

	fmt.Println("\n=== Analytics Example Complete ===")
	fmt.Println("All events have been tracked successfully!")
	fmt.Println("Note: With empty POSTHOG_PROJECT_KEY, events are not sent to PostHog (graceful degradation)")
}
