// Package analytics provides behavioral analytics and event tracking functionality
// that strictly aligns with trigger.dev's analytics.server.ts implementation.
//
// This package replicates the exact behavior of trigger.dev's BehaviouralAnalytics class:
// - PostHog integration for user behavior tracking
// - User, organization, project, environment analytics
// - Structured event capture and grouping
// - Graceful degradation when PostHog is unavailable
package analytics

import (
	"time"
)

// UserData represents user information for analytics tracking.
// Matches trigger.dev's User model properties used in analytics.
type UserData struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	Name                 string    `json:"name"`
	AuthenticationMethod string    `json:"authenticationMethod"`
	Admin                bool      `json:"admin"`
	CreatedAt            time.Time `json:"createdAt"`
}

// OrganizationData represents organization information for analytics tracking.
// Matches trigger.dev's Organization model properties used in analytics.
type OrganizationData struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ProjectData represents project information for analytics tracking.
// Matches trigger.dev's Project model properties used in analytics.
type ProjectData struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// EnvironmentData represents runtime environment information for analytics tracking.
// Matches trigger.dev's RuntimeEnvironment model properties used in analytics.
type EnvironmentData struct {
	ID             string    `json:"id"`
	Slug           string    `json:"slug"`
	OrganizationID string    `json:"organizationId"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

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
