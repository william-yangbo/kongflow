package analytics

import (
	"context"
	"fmt"
	"log"

	"github.com/posthog/posthog-go"
)

// BehaviouralAnalytics implements the AnalyticsService interface.
// This implementation strictly aligns with trigger.dev's BehaviouralAnalytics class.
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
func (b *BehaviouralAnalytics) UserIdentify(ctx context.Context, user *UserData, isNewUser bool) error {
	if b.client == nil {
		return nil // Graceful degradation when client is not available
	}

	// User identification - matches trigger.dev's client.identify() call
	err := b.client.Enqueue(posthog.Identify{
		DistinctId: user.ID,
		Properties: posthog.Properties{
			"email":                user.Email,
			"name":                 user.Name,
			"authenticationMethod": user.AuthenticationMethod,
			"admin":                user.Admin,
			"createdAt":            user.CreatedAt,
			"isNewUser":            isNewUser,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to identify user: %w", err)
	}

	// Track new user creation event - matches trigger.dev's conditional event capture
	if isNewUser {
		captureEvent := &CaptureEvent{
			UserID: user.ID,
			Event:  "user created",
			EventProperties: map[string]interface{}{
				"email":                user.Email,
				"name":                 user.Name,
				"authenticationMethod": user.AuthenticationMethod,
				"admin":                user.Admin,
				"createdAt":            user.CreatedAt,
			},
		}
		return b.capture(captureEvent)
	}

	return nil
}

// OrganizationIdentify identifies an organization for grouping.
// Matches trigger.dev's analytics.organization.identify() method.
func (b *BehaviouralAnalytics) OrganizationIdentify(ctx context.Context, org *OrganizationData) error {
	if b.client == nil {
		return nil
	}

	err := b.client.Enqueue(posthog.GroupIdentify{
		Type: "organization",
		Key:  org.ID,
		Properties: posthog.Properties{
			"name":      org.Title,
			"slug":      org.Slug,
			"createdAt": org.CreatedAt,
			"updatedAt": org.UpdatedAt,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to identify organization: %w", err)
	}

	return nil
}

// OrganizationNew tracks new organization creation.
// Matches trigger.dev's analytics.organization.new() method.
func (b *BehaviouralAnalytics) OrganizationNew(ctx context.Context, userID string, org *OrganizationData, organizationCount int) error {
	if b.client == nil {
		return nil
	}

	captureEvent := &CaptureEvent{
		UserID:         userID,
		Event:          "organization created",
		OrganizationID: &org.ID,
		EventProperties: map[string]interface{}{
			"id":        org.ID,
			"slug":      org.Slug,
			"title":     org.Title,
			"createdAt": org.CreatedAt,
			"updatedAt": org.UpdatedAt,
		},
		UserProperties: map[string]interface{}{
			"organizationCount": organizationCount,
		},
	}

	return b.capture(captureEvent)
}

// ProjectIdentify identifies a project for grouping.
// Matches trigger.dev's analytics.project.identify() method.
func (b *BehaviouralAnalytics) ProjectIdentify(ctx context.Context, project *ProjectData) error {
	if b.client == nil {
		return nil
	}

	err := b.client.Enqueue(posthog.GroupIdentify{
		Type: "project",
		Key:  project.ID,
		Properties: posthog.Properties{
			"name":      project.Name,
			"createdAt": project.CreatedAt,
			"updatedAt": project.UpdatedAt,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to identify project: %w", err)
	}

	return nil
}

// ProjectNew tracks new project creation.
// Matches trigger.dev's analytics.project.new() method.
func (b *BehaviouralAnalytics) ProjectNew(ctx context.Context, userID, organizationID string, project *ProjectData) error {
	if b.client == nil {
		return nil
	}

	captureEvent := &CaptureEvent{
		UserID:         userID,
		Event:          "project created",
		OrganizationID: &organizationID,
		EventProperties: map[string]interface{}{
			"id":        project.ID,
			"title":     project.Name,
			"createdAt": project.CreatedAt,
			"updatedAt": project.UpdatedAt,
		},
	}

	return b.capture(captureEvent)
}

// EnvironmentIdentify identifies an environment for grouping.
// Matches trigger.dev's analytics.environment.identify() method.
func (b *BehaviouralAnalytics) EnvironmentIdentify(ctx context.Context, env *EnvironmentData) error {
	if b.client == nil {
		return nil
	}

	err := b.client.Enqueue(posthog.GroupIdentify{
		Type: "environment",
		Key:  env.ID,
		Properties: posthog.Properties{
			"name":           env.Slug,
			"slug":           env.Slug,
			"organizationId": env.OrganizationID,
			"createdAt":      env.CreatedAt,
			"updatedAt":      env.UpdatedAt,
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

// capture is the internal method that handles event capture to PostHog.
// Matches trigger.dev's private #capture() method logic exactly.
func (b *BehaviouralAnalytics) capture(event *CaptureEvent) error {
	if b.client == nil {
		return nil
	}

	// Build groups - matches trigger.dev's group building logic
	groups := posthog.NewGroups()

	if event.OrganizationID != nil {
		groups.Set("organization", *event.OrganizationID)
	}

	if event.ProjectID != nil {
		groups.Set("project", *event.ProjectID)
	}

	if event.JobID != nil {
		groups.Set("workflow", *event.JobID)
	}

	if event.EnvironmentID != nil {
		groups.Set("environment", *event.EnvironmentID)
	}

	// Build properties - matches trigger.dev's property building logic
	properties := make(posthog.Properties)

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

	// Capture event - matches trigger.dev's client.capture() call
	err := b.client.Enqueue(posthog.Capture{
		DistinctId: event.UserID,
		Event:      event.Event,
		Properties: properties,
		Groups:     groups,
	})
	if err != nil {
		return fmt.Errorf("failed to capture event: %w", err)
	}

	return nil
}

// Close cleans up resources and flushes pending events.
func (b *BehaviouralAnalytics) Close() error {
	if b.client == nil {
		return nil
	}

	return b.client.Close()
}
