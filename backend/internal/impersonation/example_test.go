package impersonation_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"kongflow/backend/internal/impersonation"
)

func TestExampleUsage(t *testing.T) {
	// Example secret key (in production, use a secure random key)
	secretKey := []byte("my-secret-key-32-bytes-long!!!!!!")
	
	fmt.Println("=== Example 1: Basic Setup ===")
	
	// Create a new impersonation service
	service, err := impersonation.NewServiceWithSecretKey(secretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	// Configure for production (HTTPS)
	service.SetSecure(false) // Set to true in production with HTTPS
	
	fmt.Println("Service created successfully")
	
	fmt.Println("\n=== Example 2: Setting Impersonation ===")
	
	// Simulate an admin setting impersonation for a user
	req := httptest.NewRequest("POST", "/admin/impersonate", nil)
	w := httptest.NewRecorder()
	
	targetUserID := "user_12345"
	err = service.SetImpersonation(w, req, targetUserID)
	if err != nil {
		t.Fatalf("Failed to set impersonation: %v", err)
	}
	
	// Check that cookie was set
	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) > 0 {
		fmt.Printf("Impersonation cookie set: %s=%s...\n", 
			cookies[0].Name, cookies[0].Value[:20])
		fmt.Printf("Cookie properties: HttpOnly=%v, Secure=%v, SameSite=%v\n",
			cookies[0].HttpOnly, cookies[0].Secure, cookies[0].SameSite)
	}
	
	fmt.Println("\n=== Example 3: Checking Impersonation in Middleware ===")
	
	// Simulate a subsequent request with the impersonation cookie
	req2 := httptest.NewRequest("GET", "/api/users", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	
	// Check if user is being impersonated
	if service.IsImpersonating(req2) {
		impersonatedID, err := service.GetImpersonation(req2)
		if err != nil {
			t.Fatalf("Failed to get impersonation: %v", err)
		}
		fmt.Printf("Request is impersonating user: %s\n", impersonatedID)
	} else {
		fmt.Println("No impersonation active")
	}
	
	fmt.Println("\n=== Example 4: User ID Resolution Pattern ===")
	
	// This demonstrates the same pattern as trigger.dev's session.server.ts
	authenticatedUserID := "admin_67890"
	
	finalUserID, err := service.GetImpersonationWithFallback(req2, authenticatedUserID)
	if err != nil {
		t.Fatalf("Failed to resolve user ID: %v", err)
	}
	
	fmt.Printf("Authenticated user: %s\n", authenticatedUserID)
	fmt.Printf("Final effective user ID: %s\n", finalUserID)
	
	if finalUserID == targetUserID {
		fmt.Println("✓ Using impersonated user ID")
	} else if finalUserID == authenticatedUserID {
		fmt.Println("✓ Using authenticated user ID")
	}
	
	fmt.Println("\n=== Example 5: Clearing Impersonation ===")
	
	// Admin stops impersonating
	req3 := httptest.NewRequest("POST", "/admin/stop-impersonate", nil)
	w3 := httptest.NewRecorder()
	
	err = service.ClearImpersonation(w3, req3)
	if err != nil {
		t.Fatalf("Failed to clear impersonation: %v", err)
	}
	
	resp3 := w3.Result()
	clearCookies := resp3.Cookies()
	if len(clearCookies) > 0 && clearCookies[0].MaxAge == -1 {
		fmt.Println("Impersonation cleared successfully")
		fmt.Printf("Clear cookie set: MaxAge=%d, Value=%s\n", 
			clearCookies[0].MaxAge, clearCookies[0].Value)
	}
	
	fmt.Println("\n=== Example 6: HTTP Middleware Integration ===")
	
	// Example of how to use in HTTP middleware
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the effective user ID (impersonated or authenticated)
			authenticatedUserID := getUserFromToken(r) // Your auth logic here
			
			effectiveUserID, err := service.GetImpersonationWithFallback(r, authenticatedUserID)
			if err != nil {
				http.Error(w, "Authentication error", http.StatusUnauthorized)
				return
			}
			
			// Add user ID to request context or headers for downstream handlers
			r.Header.Set("X-User-ID", effectiveUserID)
			
			if service.IsImpersonating(r) {
				r.Header.Set("X-Impersonation-Active", "true")
				r.Header.Set("X-Original-User-ID", authenticatedUserID)
			}
			
			next.ServeHTTP(w, r)
		})
	}
	
	// Test the middleware
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		isImpersonating := r.Header.Get("X-Impersonation-Active") == "true"
		
		fmt.Printf("Handler received user ID: %s\n", userID)
		fmt.Printf("Is impersonating: %v\n", isImpersonating)
		
		if isImpersonating {
			originalUserID := r.Header.Get("X-Original-User-ID")
			fmt.Printf("Original user ID: %s\n", originalUserID)
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	
	// Test with impersonation cookie
	req4 := httptest.NewRequest("GET", "/api/test", nil)
	for _, cookie := range cookies {
		req4.AddCookie(cookie)
	}
	w4 := httptest.NewRecorder()
	
	handler.ServeHTTP(w4, req4)
	
	fmt.Println("\n=== All examples completed successfully! ===")
}

// Dummy function to simulate getting user ID from authentication token
func getUserFromToken(r *http.Request) string {
	// In a real application, this would extract the user ID from JWT token,
	// session, or other authentication mechanism
	return "admin_67890"
}

func ExampleService_SetImpersonation() {
	// Create service with secret key
	secretKey := []byte("my-secret-key-32-bytes-long!!!!!!")
	service, _ := impersonation.NewServiceWithSecretKey(secretKey)
	
	// Set up HTTP request/response
	req := httptest.NewRequest("POST", "/admin/impersonate", nil)
	w := httptest.NewRecorder()
	
	// Set impersonation for user
	err := service.SetImpersonation(w, req, "user_12345")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Println("Impersonation set successfully")
	// Output: Impersonation set successfully
}

func ExampleService_GetImpersonation() {
	secretKey := []byte("my-secret-key-32-bytes-long!!!!!!")
	service, _ := impersonation.NewServiceWithSecretKey(secretKey)
	
	// Create request with impersonation cookie
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  "__impersonate",
		Value: "signed_cookie_value_here",
	})
	
	userID, err := service.GetImpersonation(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	if userID != "" {
		fmt.Printf("Impersonating user: %s\n", userID)
	} else {
		fmt.Println("No impersonation active")
	}
}

func ExampleService_IsImpersonating() {
	secretKey := []byte("my-secret-key-32-bytes-long!!!!!!")
	service, _ := impersonation.NewServiceWithSecretKey(secretKey)
	
	req := httptest.NewRequest("GET", "/api/users", nil)
	
	if service.IsImpersonating(req) {
		fmt.Println("User is being impersonated")
	} else {
		fmt.Println("No impersonation active")
	}
	// Output: No impersonation active
}