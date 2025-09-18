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
2. **Shared Data Models**: Direct usage of SQLC-generated models from shared layer
3. **Type Aliases**: Clean API using type aliases (User, Organization, Project, Environment)
4. **Configuration**: Flexible configuration management
5. **PostHog Integration**: Backend analytics provider

### Shared Data Layer Integration

Following trigger.dev's exact architecture pattern, this implementation:

- **Uses shared data models directly**: No intermediate DTOs or adapters
- **Type aliases for clean API**: `type User = shared.Users` etc.
- **PostgreSQL type compatibility**: Handles pgtype fields (UUID, Text, Timestamptz)
- **Zero conversion overhead**: Direct mapping from database to analytics

This approach mirrors trigger.dev's `export type { User } from "@trigger.dev/database"` pattern.

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
import (
    "github.com/jackc/pgx/v5/pgtype"
    "kongflow/backend/internal/shared"
)

// Create user data using shared model with proper pgtype fields
userData := &shared.Users{
    ID:        userUUID, // pgtype.UUID
    Email:     "user@example.com", // string
    Name:      pgtype.Text{String: "John Doe", Valid: true}, // pgtype.Text
    CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
    UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
}

// Track new user registration (using type alias)
var user analytics.User = userData
err := service.UserIdentify(ctx, user, true)
```

### Organization Analytics

```go
// Create organization data using shared model
orgData := &shared.Organizations{
    ID:        orgUUID, // pgtype.UUID
    Title:     "My Company", // string
    Slug:      "my-company", // string
    CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
    UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
}

// Identify organization for grouping (using type alias)
var organization analytics.Organization = orgData
err := service.OrganizationIdentify(ctx, organization)

// Track organization creation event
err = service.OrganizationNew(ctx, userID, organization, organizationCount)
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

All data structures use shared data layer models with type aliases for trigger.dev compatibility:

- `User` (alias for `shared.Users`) ↔ trigger.dev's user interface
- `Organization` (alias for `shared.Organizations`) ↔ trigger.dev's organization interface
- `Project` (alias for `shared.Projects`) ↔ trigger.dev's project interface
- `Environment` (alias for `shared.RuntimeEnvironments`) ↔ trigger.dev's environment interface

### Shared Data Model Fields

**User Model (`shared.Users`)**:

- `ID`: pgtype.UUID
- `Email`: string
- `Name`: pgtype.Text (nullable)
- `AvatarUrl`: pgtype.Text (nullable)
- `CreatedAt`, `UpdatedAt`: pgtype.Timestamptz

**Organization Model (`shared.Organizations`)**:

- `ID`: pgtype.UUID
- `Title`, `Slug`: string
- `CreatedAt`, `UpdatedAt`: pgtype.Timestamptz

**Project Model (`shared.Projects`)**:

- `ID`, `OrganizationID`: pgtype.UUID
- `Name`, `Slug`: string
- `CreatedAt`, `UpdatedAt`: pgtype.Timestamptz

**Environment Model (`shared.RuntimeEnvironments`)**:

- `ID`, `OrganizationID`, `ProjectID`, `OrgMemberID`: pgtype.UUID
- `Slug`, `ApiKey`, `Type`: string
- `CreatedAt`, `UpdatedAt`: pgtype.Timestamptz

### PostgreSQL Type Handling

The analytics service seamlessly handles PostgreSQL-specific types:

```go
// UUID handling
userUUID := pgtype.UUID{}
userUUID.Scan("123e4567-e89b-12d3-a456-426614174000")

// Nullable text fields
name := pgtype.Text{String: "John Doe", Valid: true}
emptyName := pgtype.Text{Valid: false} // NULL value

// Timestamp handling
createdAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
```

The service automatically extracts values for PostHog integration:

- `pgtype.UUID` → `String()` method
- `pgtype.Text` → `.String` field (if `.Valid`)
- `pgtype.Timestamptz` → `.Time` field

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

This service perfectly replicates trigger.dev's architecture pattern:

### Direct Database Model Usage

Like trigger.dev's approach:

```typescript
// trigger.dev exports database types directly
export type { User, Organization } from '@trigger.dev/database';
```

Our Go implementation follows the same pattern:

```go
// Direct usage of shared data layer with type aliases
type User = shared.Users
type Organization = shared.Organizations
```

### Key Benefits

1. **Zero Abstraction Layer**: No DTOs or adapters between database and analytics
2. **Type Safety**: Direct PostgreSQL type compatibility with compile-time checking
3. **Performance**: No conversion overhead
4. **Maintainability**: Single source of truth for data structures
5. **trigger.dev Alignment**: Identical architectural philosophy

### Compatibility

This service maintains 100% behavioral compatibility with trigger.dev while leveraging:

1. Same method signatures and behaviors
2. Identical event properties and structure
3. Compatible grouping and identification
4. Matching error handling patterns
5. **Direct shared data layer usage** (just like trigger.dev)
