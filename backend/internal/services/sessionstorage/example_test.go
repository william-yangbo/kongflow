package sessionstorage_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"kongflow/backend/internal/services/sessionstorage"
)

// ExampleGetUserSession demonstrates how to retrieve user session data
func ExampleGetUserSession() {
	// Set up environment variable for testing
	os.Setenv("SESSION_SECRET", "test-secret-for-examples")

	// Create a mock request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Get user session
	session, err := sessionstorage.GetUserSession(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Set some user data
	session.Values["userID"] = "user-123"
	session.Values["username"] = "john_doe"

	// Commit session to response
	err = sessionstorage.CommitSession(req, w, session)
	if err != nil {
		fmt.Printf("Error committing session: %v\n", err)
		return
	}

	fmt.Println("Session created successfully")
	// Output: Session created successfully
}

// ExampleCommitSession demonstrates how to save session data
func ExampleCommitSession() {
	// Set up environment variable for testing
	os.Setenv("SESSION_SECRET", "test-secret-for-examples")

	// Create a mock request and response
	req := httptest.NewRequest("POST", "/login", nil)
	w := httptest.NewRecorder()

	// Get session
	session, _ := sessionstorage.GetUserSession(req)

	// Store user data after successful login
	session.Values["userID"] = "user-456"
	session.Values["role"] = "admin"
	session.Values["loginTime"] = "2024-01-15T10:30:00Z"

	// Commit the session
	err := sessionstorage.CommitSession(req, w, session)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("User logged in and session saved")
	// Output: User logged in and session saved
}

// ExampleDestroySession demonstrates how to destroy a session
func ExampleDestroySession() {
	// Set up environment variable for testing
	os.Setenv("SESSION_SECRET", "test-secret-for-examples")

	// Create a mock request with existing session
	req := httptest.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()

	// Get existing session
	session, _ := sessionstorage.GetUserSession(req)
	session.Values["userID"] = "user-789"

	// Destroy the session (logout)
	err := sessionstorage.DestroySession(req, w)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("User logged out and session destroyed")
	// Output: User logged out and session destroyed
}

// Example_usagePattern demonstrates a complete usage pattern in a web handler
func Example_usagePattern() {
	// Set up environment variable for testing
	os.Setenv("SESSION_SECRET", "test-secret-for-examples")

	// Simulate a login handler
	loginHandler := func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionstorage.GetUserSession(r)
		if err != nil {
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		// Simulate successful authentication
		userID := "user-123"
		userData := map[string]interface{}{
			"userID":   userID,
			"username": "john_doe",
			"role":     "user",
		}

		// Store user data in session
		session.Values["user"] = userData

		// Commit session
		if err := sessionstorage.CommitSession(r, w, session); err != nil {
			http.Error(w, "Failed to save session", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("Login successful"))
	}

	// Simulate a protected handler
	protectedHandler := func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionstorage.GetUserSession(r)
		if err != nil {
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		// Check if user is logged in
		userData, exists := session.Values["user"]
		if !exists {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use user data
		userDataBytes, _ := json.Marshal(userData)
		w.Header().Set("Content-Type", "application/json")
		w.Write(userDataBytes)
	}

	// Simulate a logout handler
	logoutHandler := func(w http.ResponseWriter, r *http.Request) {
		// Destroy session (DestroySession handles session retrieval internally)
		if err := sessionstorage.DestroySession(r, w); err != nil {
			http.Error(w, "Failed to destroy session", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("Logout successful"))
	}

	// Test the handlers
	fmt.Println("Handlers created successfully")

	// Prevent unused variable warnings
	_ = loginHandler
	_ = protectedHandler
	_ = logoutHandler

	// Output: Handlers created successfully
}
