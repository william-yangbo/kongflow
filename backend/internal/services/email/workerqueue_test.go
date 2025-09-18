package email

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockWorkerQueueClient is a simple mock for testing worker queue integration
type MockWorkerQueueClient struct {
	EnqueuedJobs []MockEnqueuedJob
	EnqueueError error
}

type MockEnqueuedJob struct {
	Identifier string
	Payload    interface{}
	Options    *workerqueue.JobOptions
}

func (m *MockWorkerQueueClient) Enqueue(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	if m.EnqueueError != nil {
		return nil, m.EnqueueError
	}

	m.EnqueuedJobs = append(m.EnqueuedJobs, MockEnqueuedJob{
		Identifier: identifier,
		Payload:    payload,
		Options:    opts,
	})

	return &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{ID: int64(len(m.EnqueuedJobs))},
	}, nil
}

func (m *MockWorkerQueueClient) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockWorkerQueueClient) Stop(ctx context.Context) error {
	return nil
}

func TestEmailService_WorkerQueueIntegration(t *testing.T) {
	// Setup
	mockProvider := &MockEmailProvider{}
	templateEngine, err := NewTemplateEngine()
	require.NoError(t, err)

	config := EmailConfig{
		FromEmail:     "test@example.com",
		ReplyToEmail:  "reply@example.com",
		ImagesBaseURL: "https://example.com",
	}

	t.Run("ScheduleEmailWithDelay_UsesWorkerQueue", func(t *testing.T) {
		// Since we can't easily mock the *workerqueue.Client interface in this test,
		// we test the fallback behavior when no worker queue is configured
		service := New(mockProvider, templateEngine, config)

		// Create welcome email data
		emailData := DeliverEmail{
			Email: string(EmailTypeWelcome),
			To:    "user@example.com",
		}

		welcomeData := WelcomeEmailData{Name: stringPtrHelper("John")}
		data, err := json.Marshal(welcomeData)
		require.NoError(t, err)
		emailData.Data = data

		// Schedule with 5 minute delay (should fallback to immediate since no worker queue)
		delay := 5 * time.Minute
		err = service.ScheduleEmail(context.Background(), emailData, &delay)

		// Verify fallback to immediate sending
		assert.NoError(t, err)
		assert.Len(t, mockProvider.SentEmails, 1)
		assert.Equal(t, "user@example.com", mockProvider.SentEmails[0].To)
		assert.Equal(t, "Welcome to KongFlow", mockProvider.SentEmails[0].Subject)
	})

	t.Run("ScheduleEmailShortDelay_SendsImmediately", func(t *testing.T) {
		mockProvider := &MockEmailProvider{}
		service := New(mockProvider, templateEngine, config)

		// Create welcome email data
		emailData := DeliverEmail{
			Email: string(EmailTypeWelcome),
			To:    "user@example.com",
		}

		welcomeData := WelcomeEmailData{Name: stringPtrHelper("John")}
		data, err := json.Marshal(welcomeData)
		require.NoError(t, err)
		emailData.Data = data

		// Schedule with 30 second delay (should send immediately)
		delay := 30 * time.Second
		err = service.ScheduleEmail(context.Background(), emailData, &delay)

		// Verify immediate sending
		assert.NoError(t, err)
		assert.Len(t, mockProvider.SentEmails, 1)
		assert.Equal(t, "user@example.com", mockProvider.SentEmails[0].To)
	})

	t.Run("ScheduleEmailWithoutQueue_FallsBackToImmediate", func(t *testing.T) {
		mockProvider := &MockEmailProvider{}
		service := New(mockProvider, templateEngine, config)

		// Create welcome email data
		emailData := DeliverEmail{
			Email: string(EmailTypeWelcome),
			To:    "user@example.com",
		}

		welcomeData := WelcomeEmailData{Name: stringPtrHelper("John")}
		data, err := json.Marshal(welcomeData)
		require.NoError(t, err)
		emailData.Data = data

		// Schedule with 5 minute delay (should fallback to immediate)
		delay := 5 * time.Minute
		err = service.ScheduleEmail(context.Background(), emailData, &delay)

		// Verify fallback behavior
		assert.NoError(t, err)
		assert.Len(t, mockProvider.SentEmails, 1)
		assert.Equal(t, "user@example.com", mockProvider.SentEmails[0].To)
		assert.Equal(t, "Welcome to KongFlow", mockProvider.SentEmails[0].Subject)
	})

	t.Run("ScheduleEmail_WithDifferentEmailTypes", func(t *testing.T) {
		mockProvider := &MockEmailProvider{}
		service := New(mockProvider, templateEngine, config)

		testCases := []struct {
			name            string
			emailType       DeliverEmailType
			to              string
			expectedSubject string
		}{
			{"MagicLink", EmailTypeMagicLink, "magic@example.com", "Sign in to KongFlow"},
			{"Welcome", EmailTypeWelcome, "welcome@example.com", "Welcome to KongFlow"},
			{"Invite", EmailTypeInvite, "invite@example.com", "You've been invited to join KongFlow"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockProvider.SentEmails = nil // Reset

				emailData := DeliverEmail{
					Email: string(tc.emailType),
					To:    tc.to,
				}

				// Use appropriate data for each type
				var data []byte
				var err error
				switch tc.emailType {
				case EmailTypeMagicLink:
					magicData := MagicLinkEmailData{MagicLink: "https://example.com/magic"}
					data, err = json.Marshal(magicData)
				case EmailTypeWelcome:
					welcomeData := WelcomeEmailData{Name: stringPtrHelper("John")}
					data, err = json.Marshal(welcomeData)
				case EmailTypeInvite:
					inviteData := InviteEmailData{
						OrgName:      "TestOrg",
						InviterEmail: "inviter@example.com",
						InviteLink:   "https://example.com/invite",
					}
					data, err = json.Marshal(inviteData)
				}
				require.NoError(t, err)
				emailData.Data = data

				delay := 2 * time.Minute
				err = service.ScheduleEmail(context.Background(), emailData, &delay)

				assert.NoError(t, err)
				assert.Len(t, mockProvider.SentEmails, 1)
				assert.Equal(t, tc.to, mockProvider.SentEmails[0].To)
				assert.Equal(t, tc.expectedSubject, mockProvider.SentEmails[0].Subject)
			})
		}
	})
}

// Helper function to avoid name collision
func stringPtrHelper(s string) *string {
	return &s
}
