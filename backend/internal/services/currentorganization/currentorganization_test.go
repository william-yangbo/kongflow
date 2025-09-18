package currentorganization

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Set required environment variable for tests
	os.Setenv("SESSION_SECRET", "test-secret-key-for-organization-sessions-32-chars")
	os.Setenv("NODE_ENV", "development")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Unsetenv("SESSION_SECRET")
	os.Unsetenv("NODE_ENV")

	os.Exit(code)
}

func TestGetCurrentOrg_NoCookie(t *testing.T) {
	// Test scenario: No cookie present, should return nil (like trigger.dev's undefined)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	orgSlug, err := GetCurrentOrg(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if orgSlug != nil {
		t.Errorf("Expected nil org slug when no cookie present, got %v", *orgSlug)
	}
}

func TestSetCurrentOrg(t *testing.T) {
	// Test scenario: Set organization slug and verify it's stored correctly
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Set a test organization
	testOrg := "acme-corp"
	err := SetCurrentOrg(w, req, testOrg)
	if err != nil {
		t.Fatalf("SetCurrentOrg failed: %v", err)
	}

	// Verify cookie was set
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected cookie to be set")
	}

	// Verify cookie name
	var orgCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == cookieName {
			orgCookie = cookie
			break
		}
	}
	if orgCookie == nil {
		t.Fatal("Expected __organization cookie to be set")
	}

	// Verify cookie properties match trigger.dev
	if orgCookie.Path != "/" {
		t.Errorf("Expected path '/', got '%s'", orgCookie.Path)
	}
	if orgCookie.HttpOnly != true {
		t.Error("Expected HttpOnly to be true")
	}
	if orgCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Expected SameSite Lax, got %v", orgCookie.SameSite)
	}
	// In development, Secure should be false
	if orgCookie.Secure != false {
		t.Error("Expected Secure to be false in development")
	}
}

func TestGetCurrentOrg_ValidCookie(t *testing.T) {
	// Test scenario: Set organization then retrieve it
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Set a test organization
	testOrg := "my-test-org"
	err := SetCurrentOrg(w, req, testOrg)
	if err != nil {
		t.Fatalf("SetCurrentOrg failed: %v", err)
	}

	// Create new request with the cookie from response
	newReq := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		newReq.AddCookie(cookie)
	}

	// Retrieve the organization
	retrievedOrg, err := GetCurrentOrg(newReq)
	if err != nil {
		t.Fatalf("GetCurrentOrg failed: %v", err)
	}
	if retrievedOrg == nil {
		t.Fatal("Expected organization to be retrieved, got nil")
	}

	// Verify the organization matches
	if *retrievedOrg != testOrg {
		t.Errorf("Expected org %v, got %v", testOrg, *retrievedOrg)
	}
}

func TestClearCurrentOrg(t *testing.T) {
	// Test scenario: Set organization, clear it, then verify it's gone
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Set a test organization first
	testOrg := "temp-org"
	err := SetCurrentOrg(w, req, testOrg)
	if err != nil {
		t.Fatalf("SetCurrentOrg failed: %v", err)
	}

	// Create request with the cookie
	reqWithCookie := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		reqWithCookie.AddCookie(cookie)
	}

	// Clear the current organization
	w2 := httptest.NewRecorder()
	err = ClearCurrentOrg(w2, reqWithCookie)
	if err != nil {
		t.Fatalf("ClearCurrentOrg failed: %v", err)
	}

	// Create new request with the updated cookie
	reqAfterClear := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w2.Result().Cookies() {
		reqAfterClear.AddCookie(cookie)
	}

	// Verify organization is cleared (returns nil like trigger.dev's undefined)
	clearedOrg, err := GetCurrentOrg(reqAfterClear)
	if err != nil {
		t.Fatalf("GetCurrentOrg after clear failed: %v", err)
	}
	if clearedOrg != nil {
		t.Errorf("Expected nil after clear, got %v", *clearedOrg)
	}
}

func TestClearCurrentOrg_Idempotent(t *testing.T) {
	// Test scenario: Clear organization when none is set (idempotent operation)
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Clear when no organization is set - should not error
	err := ClearCurrentOrg(w, req)
	if err != nil {
		t.Errorf("ClearCurrentOrg should be idempotent, got error: %v", err)
	}

	// Verify still no organization set
	reqAfterClear := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		reqAfterClear.AddCookie(cookie)
	}

	org, err := GetCurrentOrg(reqAfterClear)
	if err != nil {
		t.Fatalf("GetCurrentOrg failed: %v", err)
	}
	if org != nil {
		t.Errorf("Expected nil organization, got %v", *org)
	}
}

func TestSetCurrentOrg_EmptySlug(t *testing.T) {
	// Test scenario: Set empty organization slug (edge case)
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Set empty organization
	err := SetCurrentOrg(w, req, "")
	if err != nil {
		t.Fatalf("SetCurrentOrg with empty slug failed: %v", err)
	}

	// Create new request with the cookie
	newReq := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w.Result().Cookies() {
		newReq.AddCookie(cookie)
	}

	// Retrieve should return nil for empty slug
	retrievedOrg, err := GetCurrentOrg(newReq)
	if err != nil {
		t.Fatalf("GetCurrentOrg failed: %v", err)
	}
	if retrievedOrg != nil {
		t.Errorf("Expected nil for empty slug, got %v", *retrievedOrg)
	}
}

func TestOrganizationSessionRoundTrip(t *testing.T) {
	// Test scenario: Complete workflow - set, get, clear
	req := httptest.NewRequest("GET", "http://example.com", nil)

	// Step 1: User selects organization
	w1 := httptest.NewRecorder()
	selectedOrg := "enterprise-corp"
	err := SetCurrentOrg(w1, req, selectedOrg)
	if err != nil {
		t.Fatalf("Failed to set current organization: %v", err)
	}

	// Step 2: User continues with selected organization
	req2 := httptest.NewRequest("GET", "http://example.com/dashboard", nil)
	for _, cookie := range w1.Result().Cookies() {
		req2.AddCookie(cookie)
	}

	retrievedOrg, err := GetCurrentOrg(req2)
	if err != nil {
		t.Fatalf("Failed to get current organization: %v", err)
	}
	if retrievedOrg == nil || *retrievedOrg != selectedOrg {
		t.Errorf("Expected org %v, got %v", selectedOrg, retrievedOrg)
	}

	// Step 3: User switches organization (clears first)
	w3 := httptest.NewRecorder()
	err = ClearCurrentOrg(w3, req2)
	if err != nil {
		t.Fatalf("Failed to clear current organization: %v", err)
	}

	// Step 4: Verify organization is cleared
	req4 := httptest.NewRequest("GET", "http://example.com", nil)
	for _, cookie := range w3.Result().Cookies() {
		req4.AddCookie(cookie)
	}

	finalOrg, err := GetCurrentOrg(req4)
	if err != nil {
		t.Fatalf("Failed to get final organization: %v", err)
	}
	if finalOrg != nil {
		t.Errorf("Expected nil organization after clear, got %v", *finalOrg)
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

	// Manually set organization slug
	testOrg := "manual-org"
	session.Values[orgSlugKey] = testOrg

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

	retrievedOrg, err := GetCurrentOrg(req2)
	if err != nil {
		t.Fatalf("GetCurrentOrg failed: %v", err)
	}
	if retrievedOrg == nil || *retrievedOrg != testOrg {
		t.Errorf("Expected org %v, got %v", testOrg, retrievedOrg)
	}
}
