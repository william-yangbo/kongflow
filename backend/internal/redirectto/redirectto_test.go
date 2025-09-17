package redirectto

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Generate a test secret key (32 bytes for AES-256)
var testSecretKey = []byte("12345678901234567890123456789012")

func TestNewService(t *testing.T) {
	// Test with default config
	service := NewService(nil)
	if service == nil {
		t.Fatal("NewService should not return nil")
	}

	// Test with custom config
	config := DefaultConfig()
	config.SecretKey = testSecretKey
	service = NewService(config)
	if service == nil {
		t.Fatal("NewService with config should not return nil")
	}
}

func TestNewServiceWithSecretKey(t *testing.T) {
	tests := []struct {
		name      string
		secretKey []byte
		wantError bool
	}{
		{
			name:      "valid 32-byte key",
			secretKey: testSecretKey,
			wantError: false,
		},
		{
			name:      "valid 16-byte key",
			secretKey: []byte("1234567890123456"),
			wantError: false,
		},
		{
			name:      "invalid short key",
			secretKey: []byte("short"),
			wantError: true,
		},
		{
			name:      "empty key",
			secretKey: []byte{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewServiceWithSecretKey(tt.secretKey)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if service != nil {
					t.Error("Expected nil service when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if service == nil {
					t.Error("Expected service but got nil")
				}
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []string{
		"/dashboard",
		"/user/profile",
		"https://example.com/callback",
		"",
		"very long url with many parameters and special characters: /path?param1=value1&param2=value2#fragment",
	}

	for _, original := range tests {
		t.Run("encrypt_decrypt_"+original, func(t *testing.T) {
			// Encrypt
			encrypted, err := service.encrypt(original)
			if err != nil {
				t.Errorf("Encryption failed: %v", err)
				return
			}

			// Should not be the same as original
			if encrypted == original {
				t.Error("Encrypted value should be different from original")
			}

			// Decrypt
			decrypted, err := service.decrypt(encrypted)
			if err != nil {
				t.Errorf("Decryption failed: %v", err)
				return
			}

			// Should match original
			if decrypted != original {
				t.Errorf("Decrypted value %q does not match original %q", decrypted, original)
			}
		})
	}
}

func TestSetGetClearRedirectTo(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	testURL := "/dashboard"

	// Create test request and response
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Test SetRedirectTo
	err = service.SetRedirectTo(w, req, testURL)
	if err != nil {
		t.Errorf("SetRedirectTo failed: %v", err)
	}

	// Check that cookie was set
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies were set")
	}

	var redirectCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "__redirectTo" {
			redirectCookie = cookie
			break
		}
	}

	if redirectCookie == nil {
		t.Fatal("Redirect cookie was not found")
	}

	// Verify cookie properties
	if redirectCookie.HttpOnly != true {
		t.Error("Cookie should be HttpOnly")
	}
	if redirectCookie.SameSite != http.SameSiteLaxMode {
		t.Error("Cookie should have SameSite=Lax")
	}
	if redirectCookie.Path != "/" {
		t.Error("Cookie path should be /")
	}

	// Create new request with the cookie
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(redirectCookie)

	// Test GetRedirectTo
	retrievedURL, err := service.GetRedirectTo(req2)
	if err != nil {
		t.Errorf("GetRedirectTo failed: %v", err)
	}

	if retrievedURL != testURL {
		t.Errorf("Retrieved URL %q does not match original %q", retrievedURL, testURL)
	}

	// Test ClearRedirectTo
	w2 := httptest.NewRecorder()
	err = service.ClearRedirectTo(w2, req2)
	if err != nil {
		t.Errorf("ClearRedirectTo failed: %v", err)
	}

	// Check that clear cookie was set
	clearCookies := w2.Result().Cookies()
	if len(clearCookies) == 0 {
		t.Fatal("No clear cookies were set")
	}

	var clearCookie *http.Cookie
	for _, cookie := range clearCookies {
		if cookie.Name == "__redirectTo" {
			clearCookie = cookie
			break
		}
	}

	if clearCookie == nil {
		t.Fatal("Clear cookie was not found")
	}

	if clearCookie.MaxAge != -1 {
		t.Error("Clear cookie should have MaxAge=-1")
	}
}

func TestGetRedirectToNoCookie(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)

	_, err = service.GetRedirectTo(req)
	if err != ErrCookieNotFound {
		t.Errorf("Expected ErrCookieNotFound, got %v", err)
	}
}

func TestInvalidRedirectURL(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	invalidURLs := []string{
		"",
		"javascript:alert('xss')",
		"ftp://example.com",
		"data:text/html,<script>alert('xss')</script>",
		"/path\nwith\nnewlines",
		"/path\rwith\rcarriage\rreturns",
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	for _, invalidURL := range invalidURLs {
		t.Run("invalid_url_"+invalidURL, func(t *testing.T) {
			err := service.SetRedirectTo(w, req, invalidURL)
			if err != ErrInvalidRedirectURL {
				t.Errorf("Expected ErrInvalidRedirectURL for %q, got %v", invalidURL, err)
			}
		})
	}
}

func TestValidRedirectURLs(t *testing.T) {
	validURLs := []string{
		"/",
		"/dashboard",
		"/user/profile",
		"/path/to/resource?param=value",
		"/path#fragment",
		"https://example.com",
		"http://example.com",
		"https://subdomain.example.com/path?param=value#fragment",
	}

	for _, url := range validURLs {
		t.Run("valid_url_"+url, func(t *testing.T) {
			if !isValidRedirectURL(url) {
				t.Errorf("URL %q should be valid", url)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CookieName != "__redirectTo" {
		t.Errorf("Expected cookie name '__redirectTo', got %q", config.CookieName)
	}

	if config.MaxAge != 24*time.Hour {
		t.Errorf("Expected MaxAge 24h, got %v", config.MaxAge)
	}

	if config.HTTPOnly != true {
		t.Error("Expected HTTPOnly to be true")
	}

	if config.SameSite != http.SameSiteLaxMode {
		t.Error("Expected SameSite to be Lax")
	}

	if config.Path != "/" {
		t.Errorf("Expected path '/', got %q", config.Path)
	}

	if config.Secure != false {
		t.Error("Expected Secure to be false by default")
	}
}

func TestSetSecure(t *testing.T) {
	service, err := NewServiceWithSecretKey(testSecretKey)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Initially should be false
	if service.config.Secure != false {
		t.Error("Expected Secure to be false initially")
	}

	// Set to true
	service.SetSecure(true)
	if service.config.Secure != true {
		t.Error("Expected Secure to be true after SetSecure(true)")
	}

	// Set back to false
	service.SetSecure(false)
	if service.config.Secure != false {
		t.Error("Expected Secure to be false after SetSecure(false)")
	}
}
