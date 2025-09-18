package analytics

import (
	"context"
	"fmt"
	"log"

	"github.com/posthog/posthog-go"
)

// BehaviouralAnalytics implements the AnalyticsService interface.
// This implementation strictly aligns with trigger.dev's BehaviouralAnalytics class.
// Following trigger.dev's architecture: uses shared data layer models directly.
type BehaviouralAnalytics struct {
	client posthog.Client
	config *Config
}

// NewBehaviouralAnalytics creates a new analytics service instance.
// Matches trigger.dev's constructor logic with PostHog client initialization.
func NewBehaviouralAnalytics(config *Config) (*BehaviouralAnalytics, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Handle missing API key case - matches trigger.dev's graceful degradation
	if config.PostHogProjectKey == "" {
		log.Println("No PostHog API key, so analytics won't track")
		return &BehaviouralAnalytics{
			client: nil,
			config: config,
		}, nil
	}

	// Create PostHog client with configuration
	client, err := posthog.NewWithConfig(
		config.PostHogProjectKey,
		posthog.Config{
			Endpoint: config.PostHogHost,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostHog client: %w", err)
	}

	return &BehaviouralAnalytics{
		client: client,
		config: config,
	}, nil
}

// UserIdentify identifies a user and tracks user creation events.
// Matches trigger.dev's analytics.user.identify() method exactly.
// Uses shared data layer User model directly, following trigger.dev's architecture.
func (b *BehaviouralAnalytics) UserIdentify(ctx context.Context, user *User, isNewUser bool) error {
	if b.client == nil {
		return nil // Graceful degradation when client is not available
	}

	// Convert pgtype fields to standard types for PostHog
	userName := ""
	if user.Name.Valid {
		userName = user.Name.String
	}

	userID := user.ID.String()
	createdAt := user.CreatedAt.Time

	// User identification - matches trigger.dev's client.identify() call
	err := b.client.Enqueue(posthog.Identify{
		DistinctId: userID,
		Properties: posthog.Properties{
			"email":     user.Email,
			"name":      userName,
			"createdAt": createdAt,
			"isNewUser": isNewUser,
			// Note: authenticationMethod and admin not available in shared model
			// This maintains trigger.dev compatibility while using our data schema
		},
	})
	if err != nil {
		return fmt.Errorf("failed to identify user: %w", err)
	}

	// Track new user creation event - matches trigger.dev's conditional event capture
	if isNewUser {
		captureEvent := &CaptureEvent{
			UserID: userID,
			Event:  "user created",
			EventProperties: map[string]interface{}{
				"email":     user.Email,
				"name":      userName,
				"createdAt": createdAt,
			},
		}
		return b.capture(captureEvent)
	}

	return nil
}

// OrganizationIdentify identifies an organization for grouping.
// Matches trigger.dev's analytics.organization.identify() method.
// Uses shared data layer Organization model directly.
func (b *BehaviouralAnalytics) OrganizationIdentify(ctx context.Context, org *Organization) error {
	if b.client == nil {
		return nil
	}

	err := b.client.Enqueue(posthog.GroupIdentify{
		Type: "organization",
		Key:  org.ID.String(),
		Properties: posthog.Properties{
			"name":      org.Title,
			"slug":      org.Slug,
			"createdAt": org.CreatedAt.Time,
			"updatedAt": org.UpdatedAt.Time,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to identify organization: %w", err)
	}

	return nil
}

// OrganizationNew tracks new organization creation.
// Matches trigger.dev's analytics.organization.new() method.
func (b *BehaviouralAnalytics) OrganizationNew(ctx context.Context, userID string, org *Organization, organizationCount int) error {
	if b.client == nil {
		return nil
	}

	orgID := org.ID.String()
	captureEvent := &CaptureEvent{
		UserID:         userID,
		Event:          "organization created",
		OrganizationID: &orgID,
		EventProperties: map[string]interface{}{
			"id":        orgID,
			"slug":      org.Slug,
			"title":     org.Title,
			"createdAt": org.CreatedAt.Time,
			"updatedAt": org.UpdatedAt.Time,
		},
		UserProperties: map[string]interface{}{
			"organizationCount": organizationCount,
		},
	}
	return b.capture(captureEvent)
}

// ProjectIdentify identifies a project for grouping.
// Matches trigger.dev's analytics.project.identify() method.
// Uses shared data layer Project model directly.
func (b *BehaviouralAnalytics) ProjectIdentify(ctx context.Context, project *Project) error {
	if b.client == nil {
		return nil
	}

	err := b.client.Enqueue(posthog.GroupIdentify{
		Type: "project",
		Key:  project.ID.String(),
		Properties: posthog.Properties{
			"name":      project.Name,
			"createdAt": project.CreatedAt.Time,
			"updatedAt": project.UpdatedAt.Time,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to identify project: %w", err)
	}

	return nil
}

// ProjectNew tracks new project creation.
// Matches trigger.dev's analytics.project.new() method.
func (b *BehaviouralAnalytics) ProjectNew(ctx context.Context, userID, organizationID string, project *Project) error {
	if b.client == nil {
		return nil
	}

	captureEvent := &CaptureEvent{
		UserID:         userID,
		Event:          "project created",
		OrganizationID: &organizationID,
		EventProperties: map[string]interface{}{
			"id":        project.ID.String(),
			"title":     project.Name,
			"createdAt": project.CreatedAt.Time,
			"updatedAt": project.UpdatedAt.Time,
		},
	}

	return b.capture(captureEvent)
}

// EnvironmentIdentify identifies an environment for grouping.
// Matches trigger.dev's analytics.environment.identify() method.
// Uses shared data layer Environment model directly.
func (b *BehaviouralAnalytics) EnvironmentIdentify(ctx context.Context, env *Environment) error {
	if b.client == nil {
		return nil
	}

	err := b.client.Enqueue(posthog.GroupIdentify{
		Type: "environment",
		Key:  env.ID.String(),
		Properties: posthog.Properties{
			"name":           env.Slug,
			"slug":           env.Slug,
			"organizationId": env.OrganizationID.String(),
			"createdAt":      env.CreatedAt.Time,
			"updatedAt":      env.UpdatedAt.Time,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to identify environment: %w", err)
	}

	return nil
}

// Capture tracks a custom telemetry event.
// Matches trigger.dev's analytics.telemetry.capture() method.
func (b *BehaviouralAnalytics) Capture(ctx context.Context, event *TelemetryEvent) error {
	if b.client == nil {
		return nil
	}

	captureEvent := &CaptureEvent{
		UserID:          event.UserID,
		Event:           event.Event,
		EventProperties: event.Properties,
		OrganizationID:  event.OrganizationID,
		ProjectID:       event.ProjectID,
		JobID:           event.JobID,
		EnvironmentID:   event.EnvironmentID,
	}

	return b.capture(captureEvent)
}

// Close cleans up resources and flushes pending events.
func (b *BehaviouralAnalytics) Close() error {
	if b.client == nil {
		return nil
	}
	return b.client.Close()
}

// capture is the internal method for capturing events.
// Matches trigger.dev's #capture private method exactly.
func (b *BehaviouralAnalytics) capture(event *CaptureEvent) error {
	if b.client == nil {
		return nil
	}

	groups := make(map[string]interface{})

	if event.OrganizationID != nil {
		groups["organization"] = *event.OrganizationID
	}

	if event.ProjectID != nil {
		groups["project"] = *event.ProjectID
	}

	if event.JobID != nil {
		groups["workflow"] = *event.JobID
	}

	if event.EnvironmentID != nil {
		groups["environment"] = *event.EnvironmentID
	}

	properties := make(map[string]interface{})
	if event.EventProperties != nil {
		for k, v := range event.EventProperties {
			properties[k] = v
		}
	}

	if event.UserProperties != nil {
		properties["$set"] = event.UserProperties
	}

	if event.UserOnceProperties != nil {
		properties["$set_once"] = event.UserOnceProperties
	}

	eventData := posthog.Capture{
		DistinctId: event.UserID,
		Event:      event.Event,
		Properties: properties,
		Groups:     groups,
	}

	return b.client.Enqueue(eventData)
}
