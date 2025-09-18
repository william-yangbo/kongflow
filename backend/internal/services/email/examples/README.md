# Email Service Examples

This directory contains demonstration programs for the KongFlow email service.

## Available Demos

### 1. Basic Email Demo (`basic-demo/`)

Demonstrates basic email service usage with real email providers.

**Features:**

- Environment variable configuration
- Resend provider integration
- Multiple email types (welcome, notification, password reset)
- Template rendering
- Real email sending

**Run:**

```bash
cd basic-demo
go run main.go
```

**Required Environment Variables:**

- `RESEND_API_KEY` - Your Resend API key
- `FROM_EMAIL` - Sender email address
- `REPLY_TO_EMAIL` - Reply-to email address

### 2. Worker Queue Integration Demo (`worker-queue-demo/`)

Demonstrates email service integration with worker queue system.

**Features:**

- Mock email provider for testing
- Worker queue adapter integration
- Simulated worker processing
- Decoupled email sending

**Run:**

```bash
cd worker-queue-demo
go run main.go
```

**No environment variables required** - uses mock implementations.

## Architecture Notes

Both demos showcase KongFlow's clean architecture:

- **Dependency Injection**: EmailSender interface allows swapping providers
- **Loose Coupling**: Email service and worker queue are independent
- **Testability**: Mock implementations enable testing without external dependencies

Compare this with trigger.dev's tightly coupled approach that creates circular dependencies and makes testing difficult.
