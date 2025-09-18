// Package analytics provides behavioral analytics and event tracking functionality
// that strictly aligns with trigger.dev's analytics.server.ts implementation.
//
// This package replicates the exact behavior of trigger.dev's BehaviouralAnalytics class:
// - PostHog integration for user behavior tracking
// - User, organization, project, environment analytics
// - Structured event capture and grouping
// - Graceful degradation when PostHog is unavailable
//
// Following trigger.dev's architecture pattern: analytics directly uses shared data layer models
// instead of creating intermediate data transfer objects, maintaining type safety and simplicity.
package analytics

import (
	"kongflow/backend/internal/shared"
)

// Type aliases for shared data models - matches trigger.dev's approach
// These directly expose the shared data layer types for analytics use
type User = shared.Users
type Organization = shared.Organizations
type Project = shared.Projects
type Environment = shared.RuntimeEnvironments

// TelemetryEvent represents a generic event for analytics capture.
// Matches trigger.dev's telemetry.capture method parameters.
type TelemetryEvent struct {
	UserID         string                 `json:"userId"`
	Event          string                 `json:"event"`
	Properties     map[string]interface{} `json:"properties"`
	OrganizationID *string                `json:"organizationId,omitempty"`
	ProjectID      *string                `json:"projectId,omitempty"`
	JobID          *string                `json:"jobId,omitempty"`
	EnvironmentID  *string                `json:"environmentId,omitempty"`
}

// CaptureEvent represents internal event structure for PostHog capture.
// Matches trigger.dev's internal CaptureEvent type.
type CaptureEvent struct {
	UserID             string                 `json:"userId"`
	Event              string                 `json:"event"`
	OrganizationID     *string                `json:"organizationId,omitempty"`
	ProjectID          *string                `json:"projectId,omitempty"`
	JobID              *string                `json:"jobId,omitempty"`
	EnvironmentID      *string                `json:"environmentId,omitempty"`
	EventProperties    map[string]interface{} `json:"eventProperties,omitempty"`
	UserProperties     map[string]interface{} `json:"userProperties,omitempty"`
	UserOnceProperties map[string]interface{} `json:"userOnceProperties,omitempty"`
}

// Config represents the analytics service configuration.
type Config struct {
	PostHogProjectKey string
	PostHogHost       string
	Enabled           bool
}

// DefaultConfig returns the default analytics configuration.
func DefaultConfig() *Config {
	return &Config{
		PostHogHost: "https://app.posthog.com",
		Enabled:     true,
	}
}
