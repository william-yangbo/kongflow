package onboardingsession

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Set required environment variable for tests
	os.Setenv("SESSION_SECRET", "test-secret-key-for-onboarding-sessions-32-chars")
	os.Setenv("NODE_ENV", "development")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Unsetenv("SESSION_SECRET")
	os.Unsetenv("NODE_ENV")

	os.Exit(code)
}

func TestGetWorkflowDate_NoCookie(t *testing.T) {
	// Test scenario: No cookie present, should return nil (like trigger.dev's undefined)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	date, err := GetWorkflowDate(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if date != nil {
		t.Errorf("Expected nil date when no cookie present, got %v", date)
	}
}

func TestSetWorkflowDate(t *testing.T) {
	// Test scenario: Set workflow date and verify it's stored correctly
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Set a test date
	testDate := time.Date(2025, 9, 17, 14, 30, 0, 0, time.UTC)
	err := SetWorkflowDate(w, req, testDate)
	if err != nil {
		t.Fatalf("SetWorkflowDate failed: %v", err)
	}

	// Verify cookie was set
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected cookie to be set")
	}

	// Verify cookie name
	var onboardingCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == cookieName {
			onboardingCookie = cookie
			break
		}
	}
	if onboardingCookie == nil {
		t.Fatal("Expected __onboarding cookie to be set")
	}

	// Verify cookie properties match trigger.dev
	if onboardingCookie.Path != "/" {
		t.Errorf("Expected path '/', got '%s'", onboardingCookie.Path)
	}
	if onboardingCookie.HttpOnly != true {
		t.Error("Expected HttpOnly to be true")
	}
	if onboardingCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Expected SameSite Lax, got %v", onboardingCookie.SameSite)
	}
	// In development, Secure should be false
	if onboardingCookie.Secure != false {
		t.Error("Expected Secure to be false in development")
	}
}

func TestGetWorkflowDate_ValidCookie(t *testing.T) {
	// Test scenario: Set date then retrieve it
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Set a test date
	testDate := time.Date(2025, 9, 17, 14, 30, 0, 0, time.UTC)
	err := SetWorkflowDate(w, req, testDate)
	if err != nil {
		t.Fatalf("SetWorkflowDate failed: %v", err)
	}

	// Create new request with the cookie from response
	newReq := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		newReq.AddCookie(cookie)
	}

	// Retrieve the date
	retrievedDate, err := GetWorkflowDate(newReq)
	if err != nil {
		t.Fatalf("GetWorkflowDate failed: %v", err)
	}
	if retrievedDate == nil {
		t.Fatal("Expected date to be retrieved, got nil")
	}

	// Verify the date matches (allowing for timezone normalization)
	if !retrievedDate.Equal(testDate) {
		t.Errorf("Expected date %v, got %v", testDate, *retrievedDate)
	}
}

func TestClearWorkflowDate(t *testing.T) {
	// Test scenario: Set date, clear it, then verify it's gone
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Set a test date first
	testDate := time.Date(2025, 9, 17, 14, 30, 0, 0, time.UTC)
	err := SetWorkflowDate(w, req, testDate)
	if err != nil {
		t.Fatalf("SetWorkflowDate failed: %v", err)
	}

	// Create request with the cookie
	reqWithCookie := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		reqWithCookie.AddCookie(cookie)
	}

	// Clear the workflow date
	w2 := httptest.NewRecorder()
	err = ClearWorkflowDate(w2, reqWithCookie)
	if err != nil {
		t.Fatalf("ClearWorkflowDate failed: %v", err)
	}

	// Create new request with the updated cookie
	reqAfterClear := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w2.Result().Cookies() {
		reqAfterClear.AddCookie(cookie)
	}

	// Verify date is cleared (returns nil like trigger.dev's undefined)
	clearedDate, err := GetWorkflowDate(reqAfterClear)
	if err != nil {
		t.Fatalf("GetWorkflowDate after clear failed: %v", err)
	}
	if clearedDate != nil {
		t.Errorf("Expected nil after clear, got %v", clearedDate)
	}
}

func TestClearWorkflowDate_Idempotent(t *testing.T) {
	// Test scenario: Clear workflow date when none is set (idempotent operation)
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Clear when no date is set - should not error
	err := ClearWorkflowDate(w, req)
	if err != nil {
		t.Errorf("ClearWorkflowDate should be idempotent, got error: %v", err)
	}

	// Verify still no date set
	reqAfterClear := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		reqAfterClear.AddCookie(cookie)
	}

	date, err := GetWorkflowDate(reqAfterClear)
	if err != nil {
		t.Fatalf("GetWorkflowDate failed: %v", err)
	}
	if date != nil {
		t.Errorf("Expected nil date, got %v", date)
	}
}

func TestWorkflowDateRoundTrip(t *testing.T) {
	// Test scenario: Complete workflow - set, get, clear
	req := httptest.NewRequest("GET", "http://example.com", nil)

	// Step 1: Start onboarding - set workflow date
	w1 := httptest.NewRecorder()
	startDate := time.Date(2025, 9, 17, 14, 30, 0, 0, time.UTC)
	err := SetWorkflowDate(w1, req, startDate)
	if err != nil {
		t.Fatalf("Failed to set workflow date: %v", err)
	}

	// Step 2: Continue onboarding - get workflow date
	req2 := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w1.Result().Cookies() {
		req2.AddCookie(cookie)
	}

	retrievedDate, err := GetWorkflowDate(req2)
	if err != nil {
		t.Fatalf("Failed to get workflow date: %v", err)
	}
	if retrievedDate == nil || !retrievedDate.Equal(startDate) {
		t.Errorf("Expected date %v, got %v", startDate, retrievedDate)
	}

	// Step 3: Complete onboarding - clear workflow date
	w3 := httptest.NewRecorder()
	err = ClearWorkflowDate(w3, req2)
	if err != nil {
		t.Fatalf("Failed to clear workflow date: %v", err)
	}

	// Step 4: Verify onboarding is complete
	req4 := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w3.Result().Cookies() {
		req4.AddCookie(cookie)
	}

	finalDate, err := GetWorkflowDate(req4)
	if err != nil {
		t.Fatalf("Failed to get final workflow date: %v", err)
	}
	if finalDate != nil {
		t.Errorf("Expected nil date after completion, got %v", finalDate)
	}
}

func TestAdvancedAPI(t *testing.T) {
	// Test scenario: Using advanced API (GetSession + CommitSession)
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Get raw session
	session, err := GetSession(req)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	// Manually set workflow date
	testDate := time.Date(2025, 9, 17, 14, 30, 0, 0, time.UTC)
	session.Values[workflowDateKey] = testDate.Format(time.RFC3339)

	// Manually commit session
	err = CommitSession(w, req, session)
	if err != nil {
		t.Fatalf("CommitSession failed: %v", err)
	}

	// Verify using high-level API
	req2 := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		req2.AddCookie(cookie)
	}

	retrievedDate, err := GetWorkflowDate(req2)
	if err != nil {
		t.Fatalf("GetWorkflowDate failed: %v", err)
	}
	if retrievedDate == nil || !retrievedDate.Equal(testDate) {
		t.Errorf("Expected date %v, got %v", testDate, retrievedDate)
	}
}
