package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"kongflow/backend/internal/services/analytics"
	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
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

	// Create a sample UUID for the user
	userUUID := pgtype.UUID{}
	userUUID.Scan("12345678-1234-1234-1234-123456789012") // In real usage, this would be a proper UUID

	userData := &shared.Users{
		ID:        userUUID,
		Email:     "john.doe@example.com",
		Name:      pgtype.Text{String: "John Doe", Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// Track user identification (new user) - userData is already *shared.Users which is analytics.User
	err = service.UserIdentify(ctx, userData, true)
	if err != nil {
		log.Printf("Failed to identify user: %v", err)
	} else {
		fmt.Printf("✓ Identified new user: %s\n", userData.Name.String)
	}

	// Example 2: Organization creation
	fmt.Println("\n=== Organization Creation ===")

	// Create a sample UUID for the organization
	orgUUID := pgtype.UUID{}
	orgUUID.Scan("87654321-4321-4321-4321-210987654321")

	orgData := &shared.Organizations{
		ID:        orgUUID,
		Title:     "Acme Corporation",
		Slug:      "acme-corp",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// Identify organization for grouping
	err = service.OrganizationIdentify(ctx, orgData)
	if err != nil {
		log.Printf("Failed to identify organization: %v", err)
	} else {
		fmt.Printf("✓ Identified organization: %s\n", orgData.Title)
	}

	// Track organization creation event (convert UUID to string)
	err = service.OrganizationNew(ctx, userData.ID.String(), orgData, 1)
	if err != nil {
		log.Printf("Failed to track organization creation: %v", err)
	} else {
		fmt.Printf("✓ Tracked organization creation for user: %s\n", userData.Name.String)
	}

	// Example 3: Project creation
	fmt.Println("\n=== Project Creation ===")

	// Create a sample UUID for the project
	projectUUID := pgtype.UUID{}
	projectUUID.Scan("11111111-2222-3333-4444-555555555555")

	projectData := &shared.Projects{
		ID:             projectUUID,
		Name:           "My Awesome Project",
		Slug:           "my-awesome-project",
		OrganizationID: orgUUID,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// Identify project for grouping
	err = service.ProjectIdentify(ctx, projectData)
	if err != nil {
		log.Printf("Failed to identify project: %v", err)
	} else {
		fmt.Printf("✓ Identified project: %s\n", projectData.Name)
	}

	// Track project creation event (convert UUIDs to strings)
	err = service.ProjectNew(ctx, userData.ID.String(), orgData.ID.String(), projectData)
	if err != nil {
		log.Printf("Failed to track project creation: %v", err)
	} else {
		fmt.Printf("✓ Tracked project creation: %s\n", projectData.Name)
	}

	// Example 4: Environment setup
	fmt.Println("\n=== Environment Setup ===")

	// Create sample UUIDs for environment
	envUUID := pgtype.UUID{}
	envUUID.Scan("99999999-8888-7777-6666-555555555555")
	memberUUID := pgtype.UUID{}
	memberUUID.Scan("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	envData := &shared.RuntimeEnvironments{
		ID:             envUUID,
		Slug:           "production",
		ApiKey:         "env_api_key_12345",
		Type:           "PRODUCTION",
		OrganizationID: orgUUID,
		ProjectID:      projectUUID,
		OrgMemberID:    memberUUID,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
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
	orgIDStr := orgData.ID.String()
	envIDStr := envData.ID.String()

	workflowEvent := &analytics.TelemetryEvent{
		UserID: userData.ID.String(),
		Event:  "workflow_executed",
		Properties: map[string]interface{}{
			"workflow_id":  "workflow_xyz789",
			"duration_ms":  1500,
			"status":       "success",
			"trigger_type": "manual",
		},
		OrganizationID: &orgIDStr,
		EnvironmentID:  &envIDStr,
	}

	err = service.Capture(ctx, workflowEvent)
	if err != nil {
		log.Printf("Failed to capture workflow event: %v", err)
	} else {
		fmt.Printf("✓ Tracked workflow execution event\n")
	}

	// Track a feature usage event
	featureEvent := &analytics.TelemetryEvent{
		UserID: userData.ID.String(),
		Event:  "feature_used",
		Properties: map[string]interface{}{
			"feature_name": "advanced_scheduling",
			"usage_count":  5,
			"plan_type":    "pro",
		},
		OrganizationID: &orgIDStr,
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
