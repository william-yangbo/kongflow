package currentorganization_test

import (
	"fmt"
	"net/http/httptest"
	"os"

	"kongflow/backend/internal/services/currentorganization"
)

func init() {
	// Set required environment variables for examples
	os.Setenv("SESSION_SECRET", "example-secret-key-for-organization-sessions-32-chars")
	os.Setenv("NODE_ENV", "development")
}

// ExampleGetCurrentOrg demonstrates checking the current organization
func ExampleGetCurrentOrg() {
	// Create a sample HTTP request (e.g., from user visiting a page)
	req := httptest.NewRequest("GET", "http://example.com/dashboard", nil)

	// Check if user has a current organization selected
	orgSlug, err := currentorganization.GetCurrentOrg(req)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	if orgSlug == nil {
		fmt.Println("No organization selected")
	} else {
		fmt.Printf("Current organization: %s", *orgSlug)
	}

	// Output: No organization selected
}

// ExampleSetCurrentOrg demonstrates setting the current organization
func ExampleSetCurrentOrg() {
	// Create a sample HTTP request and response writer
	req := httptest.NewRequest("POST", "http://example.com/organizations/acme-corp/select", nil)
	w := httptest.NewRecorder()

	// Set the current organization when user selects it
	orgSlug := "acme-corp"
	err := currentorganization.SetCurrentOrg(w, req, orgSlug)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Organization set to: %s", orgSlug)

	// In a real application, you would also redirect or render the organization dashboard
	// w.Header().Set("Location", "/organizations/acme-corp/dashboard")
	// w.WriteHeader(http.StatusSeeOther)

	// Output: Organization set to: acme-corp
}

// ExampleClearCurrentOrg demonstrates clearing the current organization
func ExampleClearCurrentOrg() {
	// Simulate a user who has an organization set and wants to switch
	req := httptest.NewRequest("POST", "http://example.com/organizations/select", nil)
	w := httptest.NewRecorder()

	// First set an organization (simulate existing selection)
	currentorganization.SetCurrentOrg(w, req, "temp-org")

	// Create new request with the cookie from the previous response
	reqWithCookie := httptest.NewRequest("POST", "http://example.com/organizations/clear", nil)
	for _, cookie := range w.Result().Cookies() {
		reqWithCookie.AddCookie(cookie)
	}

	// Clear the current organization selection
	w2 := httptest.NewRecorder()
	err := currentorganization.ClearCurrentOrg(w2, reqWithCookie)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	fmt.Println("Organization selection cleared")

	// Output: Organization selection cleared
}

// Example_organizationSwitchingWorkflow demonstrates a complete organization switching workflow
func Example_organizationSwitchingWorkflow() {
	// Step 1: User initially has no organization selected
	req1 := httptest.NewRequest("GET", "http://example.com/dashboard", nil)

	orgSlug, err := currentorganization.GetCurrentOrg(req1)
	if err != nil {
		fmt.Printf("Error checking organization: %v", err)
		return
	}

	if orgSlug == nil {
		fmt.Println("✓ No organization selected initially")
	}

	// Step 2: User selects their primary organization
	req2 := httptest.NewRequest("POST", "http://example.com/organizations/acme-corp/select", nil)
	w2 := httptest.NewRecorder()

	err = currentorganization.SetCurrentOrg(w2, req2, "acme-corp")
	if err != nil {
		fmt.Printf("Error setting organization: %v", err)
		return
	}
	fmt.Println("✓ Selected organization: acme-corp")

	// Step 3: User navigates with selected organization (carries cookie)
	req3 := httptest.NewRequest("GET", "http://example.com/projects", nil)
	for _, cookie := range w2.Result().Cookies() {
		req3.AddCookie(cookie)
	}

	currentOrg, err := currentorganization.GetCurrentOrg(req3)
	if err != nil {
		fmt.Printf("Error getting organization: %v", err)
		return
	}

	if currentOrg != nil {
		fmt.Printf("✓ Working in organization: %s\n", *currentOrg)
	}

	// Step 4: User switches to another organization
	req4 := httptest.NewRequest("POST", "http://example.com/organizations/switch", nil)
	for _, cookie := range w2.Result().Cookies() {
		req4.AddCookie(cookie)
	}

	w4 := httptest.NewRecorder()
	err = currentorganization.SetCurrentOrg(w4, req4, "enterprise-corp")
	if err != nil {
		fmt.Printf("Error switching organization: %v", err)
		return
	}
	fmt.Println("✓ Switched to organization: enterprise-corp")

	// Step 5: Verify the switch was successful
	req5 := httptest.NewRequest("GET", "http://example.com/dashboard", nil)
	for _, cookie := range w4.Result().Cookies() {
		req5.AddCookie(cookie)
	}

	finalOrg, err := currentorganization.GetCurrentOrg(req5)
	if err != nil {
		fmt.Printf("Error final check: %v", err)
		return
	}

	if finalOrg != nil && *finalOrg == "enterprise-corp" {
		fmt.Println("✓ Organization switch confirmed")
	}

	// Output:
	// ✓ No organization selected initially
	// ✓ Selected organization: acme-corp
	// ✓ Working in organization: acme-corp
	// ✓ Switched to organization: enterprise-corp
	// ✓ Organization switch confirmed
}

// Example_multiTenantDashboard demonstrates how to use organization context in a multi-tenant app
func Example_multiTenantDashboard() {
	// Simulate a dashboard request with organization context
	req := httptest.NewRequest("GET", "http://example.com/dashboard", nil)
	w := httptest.NewRecorder()

	// First, ensure user has an organization selected (simulate middleware)
	currentorganization.SetCurrentOrg(w, req, "startup-inc")

	// Create new request with the organization cookie
	dashboardReq := httptest.NewRequest("GET", "http://example.com/dashboard", nil)
	for _, cookie := range w.Result().Cookies() {
		dashboardReq.AddCookie(cookie)
	}

	// In your dashboard handler, get the current organization
	orgSlug, err := currentorganization.GetCurrentOrg(dashboardReq)
	if err != nil {
		fmt.Printf("Error getting organization: %v", err)
		return
	}

	if orgSlug == nil {
		fmt.Println("Redirect to organization selection")
	} else {
		fmt.Printf("Rendering dashboard for organization: %s", *orgSlug)
		// Here you would:
		// 1. Load organization-specific data
		// 2. Apply organization-specific permissions
		// 3. Render organization-branded UI
	}

	// Output: Rendering dashboard for organization: startup-inc
}

// Example_advancedSessionManagement demonstrates using the lower-level session API
func Example_advancedSessionManagement() {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	// Get the raw session for advanced operations
	session, err := currentorganization.GetSession(req)
	if err != nil {
		fmt.Printf("Error getting session: %v", err)
		return
	}

	// Manually manipulate session data
	session.Values["currentOrg"] = "custom-org"
	session.Values["org_preferences"] = "dark_theme"
	session.Values["org_timezone"] = "UTC"

	// Commit the session changes
	err = currentorganization.CommitSession(w, req, session)
	if err != nil {
		fmt.Printf("Error committing session: %v", err)
		return
	}

	fmt.Println("Advanced organization session data saved")

	// Output: Advanced organization session data saved
}
