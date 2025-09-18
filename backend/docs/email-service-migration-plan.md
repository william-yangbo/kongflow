# ⚠️ Email Service Migration Plan - DEPENDENCY ISSUE IDENTIFIED

## 🚨 重要发现：依赖关系问题

经过深入分析，发现 **Email Service 依赖于 ZodWorker 队列系统**，而 ZodWorker 是一个复杂的核心基础设施组件，被大量服务使用。

### ZodWorker 依赖分析

1. **复杂度极高**: ZodWorker 基于 Graphile Worker，包含完整的任务队列系统
2. **影响范围大**: 被 20+ 个服务和路由使用，包括：

   - Email service (scheduleEmail, scheduleWelcomeEmail)
   - Run execution services
   - Event delivery services
   - Source management services
   - Job registration services

3. **迁移风险**: 需要完整的队列系统迁移，工作量巨大

## 📋 修订建议

**不推荐** Email Service 作为下一个迁移目标，建议选择更基础、独立的服务。

---

## 📋 Overview (原计划保留供参考)

This document outlines the migration plan for the Email service from trigger.dev to KongFlow backend, ensuring strict alignment with trigger.dev's implementation while adapting to Go best practices.

**Target Service**: Email Service  
**Priority**: High (Foundation service for authentication and notifications)  
**Complexity**: Medium (External service integration + queue system)  
**Dependencies**: River queue system, Resend API, configuration management

## 🎯 Migration Objectives

1. **Strict Alignment**: Replicate trigger.dev's email service behavior exactly
2. **Go Best Practices**: Implement using Go idioms and patterns
3. **Minimalist Approach**: Keep implementation simple and focused, avoid over-engineering
4. **Production Ready**: Ensure reliability for production email delivery

## 🔍 Analysis of trigger.dev Implementation

### Source Code Analysis

**File**: `apps/webapp/app/services/email.server.ts`

```typescript
import type { DeliverEmail } from 'emails';
import { EmailClient } from 'emails';
import type { SendEmailOptions } from 'remix-auth-email-link';
import { env } from '~/env.server';
import type { User } from '~/models/user.server';
import type { AuthUser } from './authUser';
import { workerQueue } from './worker.server';

const client = new EmailClient({
  apikey: env.RESEND_API_KEY,
  imagesBaseUrl: env.APP_ORIGIN,
  from: env.FROM_EMAIL,
  replyTo: env.REPLY_TO_EMAIL,
});

export async function sendMagicLinkEmail(
  options: SendEmailOptions<AuthUser>
): Promise<void> {
  return client.send({
    email: 'magic_link',
    to: options.emailAddress,
    magicLink: options.magicLink,
  });
}

export async function scheduleWelcomeEmail(user: User) {
  const delay =
    process.env.NODE_ENV === 'development' ? 1000 * 60 : 1000 * 60 * 22;
  await workerQueue.enqueue(
    'scheduleEmail',
    {
      email: 'welcome',
      to: user.email,
      name: user.name ?? undefined,
    },
    { runAt: new Date(Date.now() + delay) }
  );
}

export async function scheduleEmail(
  data: DeliverEmail,
  delay?: { seconds: number }
) {
  const runAt = delay ? new Date(Date.now() + delay.seconds * 1000) : undefined;
  await workerQueue.enqueue('scheduleEmail', data, { runAt });
}

export async function sendEmail(data: DeliverEmail) {
  return client.send(data);
}
```

### Key Characteristics

1. **Email Provider**: Uses Resend API via custom EmailClient
2. **Template System**: Supports predefined email templates ("magic_link", "welcome")
3. **Synchronous Sending**: `sendEmail()` and `sendMagicLinkEmail()` send immediately
4. **Asynchronous Scheduling**: `scheduleEmail()` and `scheduleWelcomeEmail()` use queue
5. **Configuration**: Environment-based configuration (API keys, from/reply-to addresses)
6. **Queue Integration**: Uses ZodWorker queue system for delayed email delivery

### Interface Patterns

1. **sendMagicLinkEmail(options)**: Authentication email with magic link
2. **scheduleWelcomeEmail(user)**: Welcome email with environment-based delay
3. **scheduleEmail(data, delay?)**: Generic scheduled email with optional delay
4. **sendEmail(data)**: Generic immediate email sending

### Dependencies Analysis

1. **emails package**: Custom email client and types
2. **remix-auth-email-link**: Magic link authentication types
3. **worker.server**: Queue system for scheduled emails
4. **env.server**: Environment configuration
5. **Resend API**: External email service provider

## 🏗️ Go Implementation Design

### Architecture Overview

```
kongflow/backend/
├── internal/
│   ├── services/
│   │   └── email/
│   │       ├── service.go           # Main service implementation
│   │       ├── service_test.go      # Unit tests
│   │       ├── types.go             # Type definitions
│   │       ├── templates.go         # Email template handling
│   │       ├── client.go            # Resend client wrapper
│   │       ├── client_test.go       # Client tests
│   │       └── README.md            # Service documentation
│   ├── config/
│   │   └── email.go                 # Email configuration
│   └── workers/
│       └── email_worker.go          # River job handlers
```

### Service Structure

```go
package email

import (
    "context"
    "time"
    "github.com/riverqueue/river"
)

// Service provides email functionality aligned with trigger.dev
type Service struct {
    client      EmailClient
    queueClient *river.Client[river.JobArgs]
    config      Config
}

// Config holds email service configuration
type Config struct {
    ResendAPIKey  string
    FromEmail     string
    ReplyToEmail  string
    ImagesBaseURL string
    Environment   string // "development" or "production"
}

// NewService creates a new email service
func NewService(client EmailClient, queueClient *river.Client[river.JobArgs], config Config) *Service

// Core Methods (align with trigger.dev exactly)
func (s *Service) SendMagicLinkEmail(ctx context.Context, options SendEmailOptions) error
func (s *Service) ScheduleWelcomeEmail(ctx context.Context, user User) error
func (s *Service) ScheduleEmail(ctx context.Context, data DeliverEmail, delay *time.Duration) error
func (s *Service) SendEmail(ctx context.Context, data DeliverEmail) error
```

### Type Definitions

```go
// SendEmailOptions aligns with remix-auth-email-link SendEmailOptions
type SendEmailOptions struct {
    EmailAddress string `json:"emailAddress"`
    MagicLink    string `json:"magicLink"`
}

// DeliverEmail aligns with trigger.dev DeliverEmail type
type DeliverEmail struct {
    Email string                 `json:"email"`     // Template name
    To    string                 `json:"to"`        // Recipient
    Name  *string                `json:"name"`      // Optional recipient name
    Data  map[string]interface{} `json:"data"`      // Template data
}

// User represents user data for welcome emails
type User struct {
    Email string  `json:"email"`
    Name  *string `json:"name"`
}

// EmailClient interface for testability
type EmailClient interface {
    Send(ctx context.Context, email EmailPayload) error
}

// EmailPayload for Resend API
type EmailPayload struct {
    From     string                 `json:"from"`
    To       []string               `json:"to"`
    Subject  string                 `json:"subject"`
    HTML     string                 `json:"html"`
    Text     string                 `json:"text"`
    ReplyTo  string                 `json:"reply_to"`
    Headers  map[string]string      `json:"headers"`
}
```

### River Queue Integration

```go
// EmailJobArgs for River queue
type EmailJobArgs struct {
    Type string       `json:"type"` // "welcome", "magic_link", etc.
    Data DeliverEmail `json:"data"`
}

func (EmailJobArgs) Kind() string { return "email" }

// Worker function
func EmailWorker(ctx context.Context, job *river.Job[EmailJobArgs]) error {
    // Process queued email
    return emailService.SendEmail(ctx, job.Args.Data)
}
```

## 📋 Implementation Plan

### Phase 1: Basic Email Client Implementation

**Duration**: 2 days

**Tasks**:

1. ✅ Create email service package structure
2. ✅ Implement Resend API client wrapper
3. ✅ Define core types matching trigger.dev
4. ✅ Implement basic configuration management
5. ✅ Write client unit tests

**Deliverables**:

- `internal/services/email/client.go` - Resend API integration
- `internal/services/email/types.go` - Type definitions
- `internal/config/email.go` - Configuration structure
- Unit tests for client functionality

### Phase 2: Core Service Implementation

**Duration**: 3 days

**Tasks**:

1. ✅ Implement `SendEmail()` method
2. ✅ Implement `SendMagicLinkEmail()` method
3. ✅ Create email template system
4. ✅ Add comprehensive error handling
5. ✅ Write service unit tests

**Deliverables**:

- `internal/services/email/service.go` - Main service implementation
- `internal/services/email/templates.go` - Template handling
- Unit tests with mock client
- Integration tests

### Phase 3: Queue Integration

**Duration**: 2 days

**Tasks**:

1. ✅ Implement River job types for email
2. ✅ Implement `ScheduleEmail()` method
3. ✅ Implement `ScheduleWelcomeEmail()` method
4. ✅ Create email worker for River queue
5. ✅ Test queue integration

**Deliverables**:

- `internal/workers/email_worker.go` - River job handler
- Updated service with queue methods
- Queue integration tests

### Phase 4: Configuration and Testing

**Duration**: 2 days

**Tasks**:

1. ✅ Environment-based configuration
2. ✅ Development vs production delay logic
3. ✅ Complete test coverage (>80%)
4. ✅ Integration testing with real Resend API
5. ✅ Documentation and examples

**Deliverables**:

- Comprehensive test suite
- Configuration documentation
- Service README with examples
- Integration test results

## 🔧 Technical Specifications

### Configuration

```go
type Config struct {
    ResendAPIKey  string `env:"RESEND_API_KEY" validate:"required"`
    FromEmail     string `env:"FROM_EMAIL" validate:"required,email"`
    ReplyToEmail  string `env:"REPLY_TO_EMAIL" validate:"required,email"`
    ImagesBaseURL string `env:"APP_ORIGIN" validate:"required,url"`
    Environment   string `env:"APP_ENV" default:"development"`
}
```

### Environment Variables

```bash
# Email service configuration
RESEND_API_KEY=re_xxxxxxxxx          # Resend API key
FROM_EMAIL=noreply@kongflow.com      # From address
REPLY_TO_EMAIL=support@kongflow.com  # Reply-to address
APP_ORIGIN=https://kongflow.com      # Base URL for images
APP_ENV=production                   # Environment (development/production)
```

### Template System

```go
// Template definitions
var templates = map[string]EmailTemplate{
    "magic_link": {
        Subject: "Sign in to KongFlow",
        HTMLTemplate: `<html>...</html>`,
        TextTemplate: `...`,
    },
    "welcome": {
        Subject: "Welcome to KongFlow!",
        HTMLTemplate: `<html>...</html>`,
        TextTemplate: `...`,
    },
}

type EmailTemplate struct {
    Subject      string
    HTMLTemplate string
    TextTemplate string
}
```

### Error Handling

```go
var (
    ErrInvalidEmailAddress = errors.New("invalid email address")
    ErrTemplateNotFound    = errors.New("email template not found")
    ErrAPIKeyMissing       = errors.New("resend API key is required")
    ErrSendFailed          = errors.New("failed to send email")
    ErrQueueFailed         = errors.New("failed to queue email")
)
```

## 🧪 Testing Strategy

### Unit Tests

1. **Client Tests**: Mock Resend API responses
2. **Service Tests**: Mock client and queue
3. **Template Tests**: Verify template rendering
4. **Configuration Tests**: Validate config loading

### Integration Tests

1. **Resend API Integration**: Real API calls with test email
2. **Queue Integration**: Real River queue with test jobs
3. **End-to-end Tests**: Complete email flow testing

### Test Coverage Target

- **Minimum**: 80% code coverage
- **Critical Paths**: 95+ % coverage for core methods
- **Error Scenarios**: Comprehensive error path testing

## 🚀 Deployment Considerations

### Environment Setup

1. **Development**: 1-minute email delay for testing
2. **Production**: 22-minute welcome email delay (matches trigger.dev)
3. **Testing**: Immediate delivery with test addresses

### Monitoring

1. **Email Delivery Metrics**: Success/failure rates
2. **Queue Metrics**: Job processing times and failures
3. **API Rate Limiting**: Resend API usage monitoring

### Security

1. **API Key Management**: Secure environment variable handling
2. **Email Validation**: Strict email address validation
3. **Template Security**: Prevent template injection attacks

## ✅ Acceptance Criteria

### Functional Requirements

1. ✅ `SendMagicLinkEmail()` sends authentication emails immediately
2. ✅ `ScheduleWelcomeEmail()` queues welcome emails with correct delay
3. ✅ `ScheduleEmail()` queues emails with optional custom delay
4. ✅ `SendEmail()` sends template emails immediately
5. ✅ All methods handle errors gracefully
6. ✅ Configuration loads from environment variables

### Performance Requirements

1. ✅ Email sending completes within 5 seconds
2. ✅ Queue operations complete within 1 second
3. ✅ Service startup completes within 1 second
4. ✅ Memory usage remains under 50MB

### Quality Requirements

1. ✅ Test coverage exceeds 80%
2. ✅ All public methods have documentation
3. ✅ Error messages are clear and actionable
4. ✅ Code follows Go best practices and standards

## 🚨 Risk Assessment

### High Risk

1. **Resend API Rate Limits**: Monitor usage and implement backoff
2. **Email Deliverability**: Configure SPF/DKIM records properly
3. **Queue System Failure**: Implement dead letter queues

### Medium Risk

1. **Configuration Errors**: Validate all config at startup
2. **Template Rendering Bugs**: Comprehensive template testing
3. **Memory Leaks**: Profile email processing under load

### Low Risk

1. **Minor API Changes**: Resend API is stable
2. **Template Updates**: Non-breaking template modifications

## 📚 Dependencies

### External Dependencies

1. **Resend API Client**: Official Go SDK or custom HTTP client
2. **River Queue**: Queue system for async email processing
3. **Email Templates**: HTML/text template engine

### Internal Dependencies

1. **Configuration System**: Environment variable management
2. **Logging System**: Structured logging for debugging
3. **Error Handling**: Consistent error patterns

## 📖 Documentation Requirements

1. **Service README**: Usage examples and configuration
2. **API Documentation**: Method signatures and examples
3. **Template Guide**: How to add/modify email templates
4. **Deployment Guide**: Environment setup and monitoring

---

**Document Version**: 1.0  
**Created**: September 18, 2025  
**Last Updated**: September 18, 2025  
**Status**: Ready for Implementation
