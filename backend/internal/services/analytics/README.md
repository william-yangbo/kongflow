# Analytics Service Implementation

## Overview

This package implements the `analytics` service for the kongflow backend, providing behavioral analytics and event tracking capabilities. The implementation strictly aligns with trigger.dev's `BehaviouralAnalytics` class to ensure consistency and compatibility.

## Features

- **User Identification**: Track user registration and identification
- **Organization Tracking**: Monitor organization creation and membership
- **Project Analytics**: Track project creation and usage
- **Environment Monitoring**: Monitor environment setup and configuration
- **Custom Telemetry**: Capture custom events and metrics
- **Graceful Degradation**: Functions correctly even without PostHog API key

## Architecture

### Core Components

1. **BehaviouralAnalytics**: Main service implementation
2. **Data Models**: TypeScript-aligned data structures
3. **Configuration**: Flexible configuration management
4. **PostHog Integration**: Backend analytics provider

### API Alignment

This implementation maintains 100% API compatibility with trigger.dev's analytics service:

- Method signatures match exactly
- Event properties use identical naming
- Group identification follows same patterns
- Error handling mirrors trigger.dev behavior

## Usage

### Basic Setup

```go
import "kongflow/backend/internal/services/analytics"

// Initialize with configuration
config := &analytics.Config{
    PostHogProjectKey: "your-posthog-key",
    PostHogHost:       "https://app.posthog.com",
}

service, err := analytics.NewBehaviouralAnalytics(config)
if err != nil {
    log.Fatal(err)
}
defer service.Close()
```

### User Tracking

```go
userData := &analytics.UserData{
    ID:                   "user_123",
    Email:                "user@example.com",
    Name:                 "John Doe",
    AuthenticationMethod: "email",
    Admin:                false,
    CreatedAt:            time.Now(),
}

// Track new user registration
err := service.UserIdentify(ctx, userData, true)
```

### Organization Analytics

```go
orgData := &analytics.OrganizationData{
    ID:        "org_456",
    Title:     "My Company",
    Slug:      "my-company",
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
}

// Identify organization for grouping
err := service.OrganizationIdentify(ctx, orgData)

// Track organization creation event
err = service.OrganizationNew(ctx, userID, orgData, organizationCount)
```

### Custom Events

```go
event := &analytics.TelemetryEvent{
    UserID: "user_123",
    Event:  "workflow_executed",
    Properties: map[string]interface{}{
        "workflow_id": "workflow_789",
        "duration_ms": 1500,
        "status":      "success",
    },
    OrganizationID: &orgID,
    EnvironmentID:  &envID,
}

err := service.Capture(ctx, event)
```

## Configuration

### Environment Variables

- `POSTHOG_PROJECT_KEY`: PostHog project API key (optional)
- `POSTHOG_HOST`: PostHog endpoint (defaults to app.posthog.com)

### Default Configuration

```go
config := analytics.DefaultConfig()
// Uses environment variables or safe defaults
```

## Testing

### Running Tests

```bash
go test ./internal/services/analytics/ -v
```

### Test Coverage

- ✅ Service initialization with various configurations
- ✅ User identification and event tracking
- ✅ Organization, project, and environment tracking
- ✅ Custom telemetry event capture
- ✅ Graceful degradation without API key
- ✅ Error handling and edge cases

### Example Usage

See `cmd/analytics-example/main.go` for a comprehensive usage example:

```bash
go run cmd/analytics-example/main.go
```

## PostHog Integration

### SDK Details

- **PostHog Go SDK**: v1.6.8
- **API Compatibility**: Full compatibility with PostHog's Group identification and event capture
- **Event Batching**: Automatic batching and reliable delivery
- **Error Handling**: Graceful failure handling with logging

### Event Structure

Events follow PostHog's standard format:

```json
{
  "distinct_id": "user_123",
  "event": "user created",
  "properties": {
    "email": "user@example.com",
    "name": "John Doe"
  },
  "groups": {
    "organization": "org_456",
    "project": "project_789"
  }
}
```

## Alignment with trigger.dev

### Method Mapping

| trigger.dev Method                  | Go Implementation        |
| ----------------------------------- | ------------------------ |
| `analytics.user.identify()`         | `UserIdentify()`         |
| `analytics.organization.identify()` | `OrganizationIdentify()` |
| `analytics.organization.new()`      | `OrganizationNew()`      |
| `analytics.project.identify()`      | `ProjectIdentify()`      |
| `analytics.project.new()`           | `ProjectNew()`           |
| `analytics.environment.identify()`  | `EnvironmentIdentify()`  |
| `analytics.telemetry.capture()`     | `Capture()`              |

### Property Alignment

All data structures match trigger.dev's TypeScript interfaces:

- `UserData` ↔ trigger.dev's user interface
- `OrganizationData` ↔ trigger.dev's organization interface
- `ProjectData` ↔ trigger.dev's project interface
- `EnvironmentData` ↔ trigger.dev's environment interface

## Performance

- **Asynchronous**: All events are queued and sent asynchronously
- **Batching**: PostHog SDK handles automatic batching
- **Graceful Degradation**: Zero performance impact when disabled
- **Memory Efficient**: Minimal memory footprint

## Best Practices

1. **Always use context**: Pass context for cancellation support
2. **Handle errors gracefully**: Log but don't fail critical paths
3. **Set up proper groups**: Use organization/project grouping for analytics
4. **Close service**: Always call `Close()` during shutdown
5. **Test without API key**: Ensure graceful degradation works

## Migration from trigger.dev

This service is designed as a drop-in replacement for trigger.dev's analytics:

1. Same method signatures and behaviors
2. Identical event properties and structure
3. Compatible grouping and identification
4. Matching error handling patterns

The migration maintains full behavioral compatibility while leveraging Go's performance and type safety benefits.
