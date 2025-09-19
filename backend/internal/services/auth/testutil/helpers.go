package testutil

import (
	"os"
	"time"

	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river/rivertype"
)

// SetupTestEnvironment sets up necessary environment variables for testing
func SetupTestEnvironment() {
	os.Setenv("SESSION_SECRET", "test-session-secret-for-testing")
	os.Setenv("MAGIC_LINK_SECRET", "test-magic-link-secret-for-testing")
}

// CleanupTestEnvironment cleans up test environment variables
func CleanupTestEnvironment() {
	os.Unsetenv("SESSION_SECRET")
	os.Unsetenv("MAGIC_LINK_SECRET")
}

// CreateTestUser creates a test user with the given email
func CreateTestUser(email string) *shared.Users {
	userID := pgtype.UUID{}
	userID.Scan("123e4567-e89b-12d3-a456-426614174000")

	return &shared.Users{
		ID:    userID,
		Email: email,
		Name: pgtype.Text{
			String: "Test User",
			Valid:  true,
		},
		AvatarUrl: pgtype.Text{
			String: "https://example.com/avatar.jpg",
			Valid:  true,
		},
	}
}

// MockJobInsertResult creates a mock job insert result for testing
func MockJobInsertResult() *rivertype.JobInsertResult {
	return &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:        int64(12345),
			CreatedAt: time.Now(),
			State:     "scheduled",
		},
	}
}

// CreateTestUserCreateParams creates test parameters for user creation
func CreateTestUserCreateParams(email string) shared.CreateUserParams {
	return shared.CreateUserParams{
		Email: email,
		Name: pgtype.Text{
			String: "Test User",
			Valid:  true,
		},
		AvatarUrl: pgtype.Text{
			String: "https://example.com/avatar.jpg",
			Valid:  true,
		},
	}
}

// TestUserUUID returns a standard test UUID for user IDs
func TestUserUUID() pgtype.UUID {
	userID := pgtype.UUID{}
	userID.Scan("123e4567-e89b-12d3-a456-426614174000")
	return userID
}

// TestUUID returns a UUID from a string identifier for testing
func TestUUID(identifier string) pgtype.UUID {
	// Use a mapping to ensure consistent UUIDs for the same identifier
	uuidMap := map[string]string{
		"user-123":     "123e4567-e89b-12d3-a456-426614174000",
		"new-user-123": "987fcdeb-51a2-43e1-9876-543210987654",
		"org-123":      "456789ab-cdef-1234-5678-90abcdef1234",
	}

	uuidStr, exists := uuidMap[identifier]
	if !exists {
		// Default UUID for unknown identifiers
		uuidStr = "00000000-0000-0000-0000-000000000000"
	}

	testUUID := pgtype.UUID{}
	testUUID.Scan(uuidStr)
	return testUUID
}
