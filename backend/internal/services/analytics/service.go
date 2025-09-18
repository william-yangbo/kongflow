package analytics

import (
	"context"
)

// AnalyticsService defines the interface for behavioral analytics and event tracking.
// This interface strictly aligns with trigger.dev's BehaviouralAnalytics class methods.
// Following trigger.dev's pattern: uses shared data models directly instead of custom DTOs.
type AnalyticsService interface {
	// User analytics methods
	// Matches trigger.dev's analytics.user.identify()
	UserIdentify(ctx context.Context, user *User, isNewUser bool) error

	// Organization analytics methods
	// Matches trigger.dev's analytics.organization.identify()
	OrganizationIdentify(ctx context.Context, org *Organization) error
	// Matches trigger.dev's analytics.organization.new()
	OrganizationNew(ctx context.Context, userID string, org *Organization, organizationCount int) error

	// Project analytics methods
	// Matches trigger.dev's analytics.project.identify()
	ProjectIdentify(ctx context.Context, project *Project) error
	// Matches trigger.dev's analytics.project.new()
	ProjectNew(ctx context.Context, userID, organizationID string, project *Project) error

	// Environment analytics methods
	// Matches trigger.dev's analytics.environment.identify()
	EnvironmentIdentify(ctx context.Context, env *Environment) error

	// General telemetry methods
	// Matches trigger.dev's analytics.telemetry.capture()
	Capture(ctx context.Context, event *TelemetryEvent) error

	// Internal methods
	// Close cleans up resources and flushes pending events
	Close() error
}
