package redirectto

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExampleUsage(t *testing.T) {
	// Create a secret key (in production, this should come from environment variables)
	secretKey := []byte("my-32-character-secret-key-12345") // 32 bytes for AES-256

	// Create the redirect service
	service, err := NewServiceWithSecretKey(secretKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to create service: %v", err))
	}

	// Set secure flag based on environment (true in production)
	service.SetSecure(false) // false for development

	// Example 1: Setting a redirect URL
	fmt.Println("=== Example 1: Setting Redirect URL ===")

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()

	targetURL := "/dashboard"
	err = service.SetRedirectTo(w, req, targetURL)
	if err != nil {
		fmt.Printf("Error setting redirect: %v\n", err)
		return
	}

	fmt.Printf("Successfully set redirect URL: %s\n", targetURL)

	// Get the cookie that was set
	cookies := w.Result().Cookies()
	var redirectCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "__redirectTo" {
			redirectCookie = cookie
			break
		}
	}

	if redirectCookie != nil {
		fmt.Printf("Cookie set: %s=%s\n", redirectCookie.Name, redirectCookie.Value[:20]+"...")
		fmt.Printf("Cookie properties: HttpOnly=%v, Secure=%v, SameSite=%v\n",
			redirectCookie.HttpOnly, redirectCookie.Secure, redirectCookie.SameSite)
	}

	// Example 2: Getting the redirect URL
	fmt.Println("\n=== Example 2: Getting Redirect URL ===")

	// Create a new request with the cookie
	req2 := httptest.NewRequest("GET", "/auth/callback", nil)
	if redirectCookie != nil {
		req2.AddCookie(redirectCookie)
	}

	retrievedURL, err := service.GetRedirectTo(req2)
	if err != nil {
		fmt.Printf("Error getting redirect: %v\n", err)
		return
	}

	fmt.Printf("Retrieved redirect URL: %s\n", retrievedURL)
	fmt.Printf("URLs match: %v\n", retrievedURL == targetURL)

	// Example 3: Clearing the redirect URL
	fmt.Println("\n=== Example 3: Clearing Redirect URL ===")

	w3 := httptest.NewRecorder()
	err = service.ClearRedirectTo(w3, req2)
	if err != nil {
		fmt.Printf("Error clearing redirect: %v", err)
		return
	}

	fmt.Println("Successfully cleared redirect URL")

	// Verify cookie was cleared
	clearCookies := w3.Result().Cookies()
	for _, cookie := range clearCookies {
		if cookie.Name == "__redirectTo" {
			fmt.Printf("Clear cookie set: MaxAge=%d, Value=%s\n", cookie.MaxAge, cookie.Value)
			break
		}
	}

	// Example 4: Error handling - no cookie
	fmt.Println("\n=== Example 4: Error Handling ===")

	reqNoCookie := httptest.NewRequest("GET", "/", nil)
	_, err = service.GetRedirectTo(reqNoCookie)
	if err != nil {
		fmt.Printf("Expected error when no cookie present: %v\n", err)
	}

	// Example 5: Invalid URLs
	fmt.Println("\n=== Example 5: URL Validation ===")

	invalidURLs := []string{
		"javascript:alert('xss')",
		"",
		"ftp://example.com",
	}

	for _, invalidURL := range invalidURLs {
		err = service.SetRedirectTo(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil), invalidURL)
		if err != nil {
			fmt.Printf("Correctly rejected invalid URL '%s': %v\n", invalidURL, err)
		}
	}

	fmt.Println("\n=== All examples completed successfully! ===")
}
