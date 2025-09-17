package redirectto

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAlignedServiceBehavior tests that our Go implementation exactly matches trigger.dev behavior
func TestAlignedServiceBehavior(t *testing.T) {
	secretKey := []byte("12345678901234567890123456789012") // 32 bytes
	
	config := DefaultConfig()
	config.SecretKey = secretKey
	service := NewAlignedService(config)
	
	t.Run("exact_trigger_dev_workflow", func(t *testing.T) {
		// Simulate the exact trigger.dev workflow
		
		// 1. getRedirectSession (should return empty session when no cookie)
		req := httptest.NewRequest("GET", "/login", nil)
		session, err := service.GetRedirectSession(req)
		if err != nil {
			t.Fatalf("GetRedirectSession failed: %v", err)
		}
		if session == nil {
			t.Fatal("GetRedirectSession should never return nil session (like Remix)")
		}
		
		// 2. setRedirectTo (should modify session and return it)
		redirectURL := "/dashboard"
		session, err = service.SetRedirectTo(req, redirectURL)
		if err != nil {
			t.Fatalf("SetRedirectTo failed: %v", err)
		}
		if session == nil {
			t.Fatal("SetRedirectTo should return session object")
		}
		
		// Verify session contains the redirect URL
		storedValue := session.Get("redirectTo")
		if storedValue != redirectURL {
			t.Errorf("Expected stored value %q, got %v", redirectURL, storedValue)
		}
		
		// 3. commitSession (should return Set-Cookie header)
		cookieHeader, err := service.CommitSession(session)
		if err != nil {
			t.Fatalf("CommitSession failed: %v", err)
		}
		if cookieHeader == "" {
			t.Fatal("CommitSession should return cookie header when session is dirty")
		}
		
		// Verify cookie format
		if !strings.Contains(cookieHeader, "__redirectTo=") {
			t.Error("Cookie header should contain __redirectTo cookie")
		}
		if !strings.Contains(cookieHeader, "HttpOnly") {
			t.Error("Cookie should be HttpOnly")
		}
		if !strings.Contains(cookieHeader, "SameSite=Lax") {
			t.Error("Cookie should have SameSite=Lax")
		}
		
		// 4. Parse cookie and create new request (simulate browser behavior)
		// Extract cookie value from header
		parts := strings.Split(cookieHeader, ";")
		var cookieValue string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "__redirectTo=") {
				cookieValue = strings.TrimPrefix(part, "__redirectTo=")
				break
			}
		}
		if cookieValue == "" {
			t.Fatal("Could not extract cookie value from header")
		}
		
		// 5. New request with cookie (simulate auth callback)
		req2 := httptest.NewRequest("GET", "/auth/callback", nil)
		req2.AddCookie(&http.Cookie{
			Name:  "__redirectTo",
			Value: cookieValue,
		})
		
		// 6. getRedirectTo (should return the stored URL)
		retrievedURL, err := service.GetRedirectTo(req2)
		if err != nil {
			t.Fatalf("GetRedirectTo failed: %v", err)
		}
		
		// Check exact trigger.dev behavior: returns *string (can be nil)
		if retrievedURL == nil {
			t.Fatal("GetRedirectTo should return non-nil string pointer")
		}
		if *retrievedURL != redirectURL {
			t.Errorf("Expected retrieved URL %q, got %q", redirectURL, *retrievedURL)
		}
		
		// 7. clearRedirectTo (should remove the value)
		clearedSession, err := service.ClearRedirectTo(req2)
		if err != nil {
			t.Fatalf("ClearRedirectTo failed: %v", err)
		}
		if clearedSession == nil {
			t.Fatal("ClearRedirectTo should return session object")
		}
		
		// Verify value was removed
		if clearedSession.Has("redirectTo") {
			t.Error("redirectTo should be removed from session")
		}
		
		// 8. Commit cleared session
		clearCookieHeader, err := service.CommitSession(clearedSession)
		if err != nil {
			t.Fatalf("CommitSession for cleared session failed: %v", err)
		}
		
		// Should still return a cookie header (to update the browser)
		if clearCookieHeader == "" {
			t.Error("CommitSession should return header even for cleared session")
		}
	})
	
	t.Run("no_cookie_behavior", func(t *testing.T) {
		// Test trigger.dev behavior when no cookie exists
		req := httptest.NewRequest("GET", "/", nil)
		
		redirectURL, err := service.GetRedirectTo(req)
		if err != nil {
			t.Errorf("GetRedirectTo should not error when no cookie: %v", err)
		}
		
		// Should return nil (undefined in TypeScript)
		if redirectURL != nil {
			t.Error("GetRedirectTo should return nil when no cookie exists")
		}
	})
	
	t.Run("invalid_cookie_behavior", func(t *testing.T) {
		// Test trigger.dev behavior with invalid cookie
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "__redirectTo",
			Value: "invalid-cookie-data",
		})
		
		redirectURL, err := service.GetRedirectTo(req)
		if err != nil {
			t.Errorf("GetRedirectTo should not error with invalid cookie: %v", err)
		}
		
		// Should return nil (like Remix does with invalid sessions)
		if redirectURL != nil {
			t.Error("GetRedirectTo should return nil for invalid cookie")
		}
	})
}

func TestRemixSessionBehavior(t *testing.T) {
	secretKey := []byte("12345678901234567890123456789012")
	config := DefaultConfig()
	config.SecretKey = secretKey
	service := NewAlignedService(config)
	
	t.Run("session_api_compatibility", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		
		session, err := service.GetSession(req)
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}
		
		// Test Remix-like session API
		session.Set("key1", "value1")
		session.Set("key2", 42)
		session.Set("key3", true)
		
		if !session.Has("key1") {
			t.Error("Session should have key1")
		}
		
		if session.Get("key1") != "value1" {
			t.Error("Session should return correct value for key1")
		}
		
		session.Unset("key2")
		if session.Has("key2") {
			t.Error("Session should not have key2 after unset")
		}
		
		// Test dirty flag
		if !session.dirty {
			t.Error("Session should be dirty after modifications")
		}
	})
}

// Performance test to ensure signing/verification is efficient
func BenchmarkCookieSigning(b *testing.B) {
	secretKey := []byte("12345678901234567890123456789012")
	config := DefaultConfig()
	config.SecretKey = secretKey
	service := NewAlignedService(config)
	
	testData := `{"redirectTo":"/dashboard","__id":"abc123","__expires":1234567890}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signed, err := service.signCookie(testData)
		if err != nil {
			b.Fatal(err)
		}
		
		_, err = service.unsignCookie(signed)
		if err != nil {
			b.Fatal(err)
		}
	}
}