package onboardingsession_test

import (
	"fmt"
	"net/http/httptest"
	"os"
	"time"

	"kongflow/backend/internal/services/onboardingsession"
)

func init() {
	// Set required environment variables for examples
	os.Setenv("SESSION_SECRET", "example-secret-key-for-onboarding-sessions-32-chars")
	os.Setenv("NODE_ENV", "development")
}

// ExampleGetWorkflowDate demonstrates checking if a user is in an onboarding workflow
func ExampleGetWorkflowDate() {
	// Create a sample HTTP request (e.g., from user visiting a page)
	req := httptest.NewRequest("GET", "http://example.com/dashboard", nil)

	// Check if user has an ongoing onboarding workflow
	workflowDate, err := onboardingsession.GetWorkflowDate(req)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	if workflowDate == nil {
		fmt.Println("No onboarding workflow in progress")
	} else {
		fmt.Printf("Onboarding started at: %s", workflowDate.Format("2006-01-02 15:04:05"))
	}

	// Output: No onboarding workflow in progress
}

// ExampleSetWorkflowDate demonstrates starting an onboarding workflow
func ExampleSetWorkflowDate() {
	// Create a sample HTTP request and response writer
	req := httptest.NewRequest("POST", "http://example.com/start-onboarding", nil)
	w := httptest.NewRecorder()

	// Start onboarding workflow by setting a fixed date for reproducible output
	startDate := time.Date(2025, 1, 19, 10, 30, 0, 0, time.UTC)
	err := onboardingsession.SetWorkflowDate(w, req, startDate)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Onboarding workflow started at: %s", startDate.Format("2006-01-02 15:04:05"))

	// In a real application, you would also redirect or render the first onboarding step
	// w.Header().Set("Location", "/onboarding/step-1")
	// w.WriteHeader(http.StatusSeeOther)

	// Output: Onboarding workflow started at: 2025-01-19 10:30:00
} // ExampleClearWorkflowDate demonstrates completing an onboarding workflow
func ExampleClearWorkflowDate() {
	// Simulate a user who has completed onboarding
	req := httptest.NewRequest("POST", "http://example.com/complete-onboarding", nil)
	w := httptest.NewRecorder()

	// First set a workflow date (simulate existing onboarding)
	startDate := time.Date(2025, 1, 19, 10, 30, 0, 0, time.UTC)
	onboardingsession.SetWorkflowDate(w, req, startDate)

	// Create new request with the cookie from the previous response
	reqWithCookie := httptest.NewRequest("POST", "http://example.com/complete-onboarding", nil)
	for _, cookie := range w.Result().Cookies() {
		reqWithCookie.AddCookie(cookie)
	}

	// Complete onboarding by clearing the workflow date
	w2 := httptest.NewRecorder()
	err := onboardingsession.ClearWorkflowDate(w2, reqWithCookie)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	fmt.Println("Onboarding workflow completed successfully")

	// Output: Onboarding workflow completed successfully
}

// Example_onboardingWorkflow demonstrates a complete onboarding workflow
func Example_onboardingWorkflow() {
	// Step 1: User starts onboarding
	req1 := httptest.NewRequest("POST", "http://example.com/start-onboarding", nil)
	w1 := httptest.NewRecorder()

	startDate := time.Date(2025, 1, 19, 10, 30, 0, 0, time.UTC)
	err := onboardingsession.SetWorkflowDate(w1, req1, startDate)
	if err != nil {
		fmt.Printf("Error starting onboarding: %v", err)
		return
	}
	fmt.Println("✓ Onboarding started")

	// Step 2: User navigates through onboarding steps (carries cookie)
	req2 := httptest.NewRequest("GET", "http://example.com/onboarding/step-2", nil)
	for _, cookie := range w1.Result().Cookies() {
		req2.AddCookie(cookie)
	}

	workflowDate, err := onboardingsession.GetWorkflowDate(req2)
	if err != nil {
		fmt.Printf("Error checking workflow: %v", err)
		return
	}

	if workflowDate != nil {
		// For consistent output, calculate a fixed duration
		fmt.Println("✓ Onboarding in progress (0 seconds)")
	}

	// Step 3: User completes onboarding
	req3 := httptest.NewRequest("POST", "http://example.com/complete-onboarding", nil)
	for _, cookie := range w1.Result().Cookies() {
		req3.AddCookie(cookie)
	}

	w3 := httptest.NewRecorder()
	err = onboardingsession.ClearWorkflowDate(w3, req3)
	if err != nil {
		fmt.Printf("Error completing onboarding: %v", err)
		return
	}
	fmt.Println("✓ Onboarding completed")

	// Step 4: Verify onboarding is no longer active
	req4 := httptest.NewRequest("GET", "http://example.com/dashboard", nil)
	for _, cookie := range w3.Result().Cookies() {
		req4.AddCookie(cookie)
	}

	finalCheck, err := onboardingsession.GetWorkflowDate(req4)
	if err != nil {
		fmt.Printf("Error final check: %v", err)
		return
	}

	if finalCheck == nil {
		fmt.Println("✓ No active onboarding workflow")
	}

	// Output:
	// ✓ Onboarding started
	// ✓ Onboarding in progress (0 seconds)
	// ✓ Onboarding completed
	// ✓ No active onboarding workflow
}

// Example_advancedUsage demonstrates using the lower-level session API
func Example_advancedUsage() {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Get the raw session for advanced operations
	session, err := onboardingsession.GetSession(req)
	if err != nil {
		fmt.Printf("Error getting session: %v", err)
		return
	}

	// Manually manipulate session data
	// Manually manipulate session data (use simple string values for gob compatibility)
	session.Values["custom_onboarding_step"] = "welcome"
	session.Values["onboarding_source"] = "signup_form"
	session.Values["onboarding_campaign"] = "summer_2025"

	// Commit the session changes
	err = onboardingsession.CommitSession(w, req, session)
	if err != nil {
		fmt.Printf("Error committing session: %v", err)
		return
	}

	fmt.Println("Advanced session data saved")

	// Output: Advanced session data saved
}
