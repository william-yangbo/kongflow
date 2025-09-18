package analytics

import (
	"context"
)

// AnalyticsService defines the interface for behavioral analytics and event tracking.
// This interface strictly aligns with trigger.dev's BehaviouralAnalytics class methods.
type AnalyticsService interface {
	// User analytics methods
	// Matches trigger.dev's analytics.user.identify()
	UserIdentify(ctx context.Context, user *UserData, isNewUser bool) error

	// Organization analytics methods
	// Matches trigger.dev's analytics.organization.identify()
	OrganizationIdentify(ctx context.Context, org *OrganizationData) error
	// Matches trigger.dev's analytics.organization.new()
	OrganizationNew(ctx context.Context, userID string, org *OrganizationData, organizationCount int) error

	// Project analytics methods
	// Matches trigger.dev's analytics.project.identify()
	ProjectIdentify(ctx context.Context, project *ProjectData) error
	// Matches trigger.dev's analytics.project.new()
	ProjectNew(ctx context.Context, userID, organizationID string, project *ProjectData) error

	// Environment analytics methods
	// Matches trigger.dev's analytics.environment.identify()
	EnvironmentIdentify(ctx context.Context, env *EnvironmentData) error

	// General telemetry methods
	// Matches trigger.dev's analytics.telemetry.capture()
	Capture(ctx context.Context, event *TelemetryEvent) error

	// Internal methods
	// Close cleans up resources and flushes pending events
	Close() error
}
