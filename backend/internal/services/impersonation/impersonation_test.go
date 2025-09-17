package impersonation

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testSecretKey = []byte("test-secret-key-32-bytes-long!!!")

func TestNewService(t *testing.T) {
	// Test with nil config
	service := NewService(nil)
	if service == nil {
		t.Fatal("NewService should not return nil")
	}

	if service.config.CookieName != "__impersonate" {
		t.Errorf("Expected cookie name '__impersonate', got %s", service.config.CookieName)
	}

	// Test with custom config
	config := &Config{
		CookieName: "custom_impersonate",
		SecretKey:  testSecretKey,
		MaxAge:     time.Hour,
	}

	service = NewService(config)
	if service.config.CookieName != "custom_impersonate" {
		t.Errorf("Expected cookie name 'custom_impersonate', got %s", service.config.CookieName)
	}
}

func TestNewServiceWithSecretKey(t *testing.T) {
	tests := []struct {
		name      string
		secretKey []byte
		wantError bool
	}{
		{
			name:      "valid_32-byte_key",
			secretKey: testSecretKey,
			wantError: false,
		},
		{
			name:      "valid_16-byte_key",
			secretKey: []byte("sixteen-byte-key"),
			wantError: false,
		},
		{
			name:      "invalid_short_key",
			secretKey: []byte("short"),
			wantError: true,
		},
		{
			name:      "empty_key",
			secretKey: []byte{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewServiceWithSecretKey(tt.secretKey)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for %s, got none", tt.name)
				}
				if service != nil {
					t.Errorf("Expected nil service for error case, got %v", service)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
				if service == nil {
					t.Errorf("Expected service for %s, got nil", tt.name)
				}
			}
		})
	}
}

func TestSetGetClearImpersonation(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Create test request and response
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	testUserID := "user123"

	// Test setting impersonation
	err = service.SetImpersonation(w, req, testUserID)
	if err != nil {
		t.Fatalf("Failed to set impersonation: %v", err)
	}

	// Check that cookie was set
	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies were set")
	}

	var impersonationCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "__impersonate" {
			impersonationCookie = cookie
			break
		}
	}

	if impersonationCookie == nil {
		t.Fatal("Impersonation cookie not found")
	}

	// Test getting impersonation from a new request with the cookie
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(impersonationCookie)

	retrievedUserID, err := service.GetImpersonation(req2)
	if err != nil {
		t.Fatalf("Failed to get impersonation: %v", err)
	}

	if retrievedUserID != testUserID {
		t.Errorf("Expected user ID %s, got %s", testUserID, retrievedUserID)
	}

	// Test IsImpersonating
	if !service.IsImpersonating(req2) {
		t.Error("IsImpersonating should return true")
	}

	// Test clearing impersonation
	w2 := httptest.NewRecorder()
	err = service.ClearImpersonation(w2, req2)
	if err != nil {
		t.Fatalf("Failed to clear impersonation: %v", err)
	}

	// Check that clear cookie was set
	resp2 := w2.Result()
	clearCookies := resp2.Cookies()
	if len(clearCookies) == 0 {
		t.Fatal("No clear cookies were set")
	}

	clearCookie := clearCookies[0]
	if clearCookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1 for clear cookie, got %d", clearCookie.MaxAge)
	}
}

func TestGetImpersonationNoCookie(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)

	userID, err := service.GetImpersonation(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if userID != "" {
		t.Errorf("Expected empty user ID, got %s", userID)
	}

	if service.IsImpersonating(req) {
		t.Error("IsImpersonating should return false when no cookie")
	}
}

func TestInvalidUserID(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	tests := []string{"", "   ", "\t\n"}

	for _, invalidUserID := range tests {
		err = service.SetImpersonation(w, req, invalidUserID)
		if err != ErrInvalidUserID {
			t.Errorf("Expected ErrInvalidUserID for '%s', got %v", invalidUserID, err)
		}
	}
}

func TestInvalidSecretKey(t *testing.T) {
	service := NewService(&Config{
		CookieName: "__impersonate",
		// No secret key set
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	err := service.SetImpersonation(w, req, "user123")
	if err != ErrInvalidSecretKey {
		t.Errorf("Expected ErrInvalidSecretKey, got %v", err)
	}
}

func TestCookieSignatureValidation(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)

	// Add a cookie with invalid signature
	invalidCookie := &http.Cookie{
		Name:  "__impersonate",
		Value: "dGVzdA==.invalid_signature",
	}
	req.AddCookie(invalidCookie)

	userID, err := service.GetImpersonation(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return empty string for invalid signature
	if userID != "" {
		t.Errorf("Expected empty user ID for invalid signature, got %s", userID)
	}
}

func TestGetImpersonationWithFallback(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	fallbackUserID := "fallback123"

	// Test without impersonation
	req := httptest.NewRequest("GET", "/", nil)
	userID, err := service.GetImpersonationWithFallback(req, fallbackUserID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if userID != fallbackUserID {
		t.Errorf("Expected fallback user ID %s, got %s", fallbackUserID, userID)
	}

	// Test with impersonation
	w := httptest.NewRecorder()
	impersonatedUserID := "impersonated456"

	err = service.SetImpersonation(w, req, impersonatedUserID)
	if err != nil {
		t.Fatalf("Failed to set impersonation: %v", err)
	}

	// Create new request with the cookie
	resp := w.Result()
	cookies := resp.Cookies()
	req2 := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}

	userID, err = service.GetImpersonationWithFallback(req2, fallbackUserID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if userID != impersonatedUserID {
		t.Errorf("Expected impersonated user ID %s, got %s", impersonatedUserID, userID)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CookieName != "__impersonate" {
		t.Errorf("Expected cookie name '__impersonate', got %s", config.CookieName)
	}

	if config.Path != "/" {
		t.Errorf("Expected path '/', got %s", config.Path)
	}

	if config.MaxAge != 24*time.Hour {
		t.Errorf("Expected MaxAge 24h, got %v", config.MaxAge)
	}

	if !config.HttpOnly {
		t.Error("Expected HttpOnly to be true")
	}

	if config.SameSite != http.SameSiteLaxMode {
		t.Errorf("Expected SameSite Lax, got %v", config.SameSite)
	}

	if config.Secure {
		t.Error("Expected Secure to be false by default")
	}
}

func TestSetSecure(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test setting secure to true
	service.SetSecure(true)
	if !service.config.Secure {
		t.Error("Expected Secure to be true after SetSecure(true)")
	}

	// Test setting secure to false
	service.SetSecure(false)
	if service.config.Secure {
		t.Error("Expected Secure to be false after SetSecure(false)")
	}
}
