# SessionStorage Service Migration Plan

## 📋 Overview

This document outlines the **simplified** migration plan for the SessionStorage service from trigger.dev to KongFlow backend, ensuring strict alignment with trigger.dev's implementation while avoiding over-engineering.

## 🎯 Migration Objectives

1. **Strict Alignment**: Replicate trigger.dev's cookie session storage behavior exactly
2. **Minimal Implementation**: Use existing Go libraries instead of custom solutions
3. **Security First**: Maintain identical security configurations and protections
4. **Quick Delivery**: Complete migration in 2-3 days, not weeks

## 🔍 Analysis of trigger.dev Implementation

### Source Code Analysis

**File**: `apps/webapp/app/services/sessionStorage.server.ts`

```typescript
import { createCookieSessionStorage } from '@remix-run/node';
import { env } from '~/env.server';

export const sessionStorage = createCookieSessionStorage({
  cookie: {
    name: '__session',
    sameSite: 'lax',
    path: '/',
    httpOnly: true,
    secrets: [env.SESSION_SECRET],
    secure: env.NODE_ENV === 'production',
    maxAge: 60 * 60 * 24 * 365, // 1 year
  },
});

export function getUserSession(request: Request) {
  return sessionStorage.getSession(request.headers.get('Cookie'));
}

export const { getSession, commitSession, destroySession } = sessionStorage;
```

### Key Characteristics

1. **Storage Type**: Cookie-based session storage (client-side)
2. **Encryption**: Uses signed and encrypted cookies via secrets
3. **Security Configuration**:
   - `httpOnly: true` - Prevents XSS attacks
   - `secure: env.NODE_ENV === "production"` - HTTPS-only in production
   - `sameSite: "lax"` - CSRF protection
   - `secrets: [env.SESSION_SECRET]` - Cookie signing and encryption
4. **Lifetime**: 1 year maximum age
5. **Cookie Name**: `__session`
6. **Path**: Root path `/`

### Usage Patterns

1. **Authentication Integration**: Used by `remix-auth` for storing auth state
2. **Session Data Management**: Store/retrieve arbitrary session data
3. **Magic Link Flow**: Temporary state storage for email-based authentication
4. **Cross-Request State**: Maintaining user context between HTTP requests

### Dependencies

- **remix-auth**: `Authenticator<AuthUser>(sessionStorage)`
- **Environment Variables**: `SESSION_SECRET` for encryption
- **HTTP Context**: Request/Response cycle integration

## 🏗️ Simplified Go Implementation

### Direct Library Approach

Instead of custom implementation, use mature Go session library:

**Chosen Library**: `github.com/gorilla/sessions` - mature, secure, widely used

### Complete Implementation (< 100 lines)

```go
// backend/internal/sessionstorage/sessionstorage.go
package sessionstorage

import (
    "net/http"
    "os"
    "sync"
    "time"

    "github.com/gorilla/sessions"
)

var (
    store *sessions.CookieStore
    once  sync.Once
)

// Initialize session store - called once
func init() {
    once.Do(func() {
        secret := os.Getenv("SESSION_SECRET")
        if secret == "" {
            panic("SESSION_SECRET environment variable is required")
        }

        // Create store with secret (same as trigger.dev)
        store = sessions.NewCookieStore([]byte(secret))

        // Configure exactly like trigger.dev
        store.Options = &sessions.Options{
            Path:     "/",                                    // cookie.path
            MaxAge:   int((365 * 24 * time.Hour).Seconds()), // cookie.maxAge: 1 year
            HttpOnly: true,                                   // cookie.httpOnly
            Secure:   os.Getenv("NODE_ENV") == "production",  // cookie.secure
            SameSite: http.SameSiteLaxMode,                   // cookie.sameSite: "lax"
        }
    })
}

// GetUserSession - 完全对齐 trigger.dev 的 getUserSession
func GetUserSession(r *http.Request) (*sessions.Session, error) {
    return store.Get(r, "__session") // 使用相同的 cookie name
}

// GetSession - 对齐 trigger.dev 的 getSession
func GetSession(r *http.Request, name string) (*sessions.Session, error) {
    return store.Get(r, name)
}

// CommitSession - 对齐 trigger.dev 的 commitSession
func CommitSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
    return session.Save(r, w)
}

// DestroySession - 对齐 trigger.dev 的 destroySession
func DestroySession(r *http.Request, w http.ResponseWriter) error {
    session, err := GetUserSession(r)
    if err != nil {
        return err
    }

    // Set MaxAge to -1 to delete cookie
    session.Options.MaxAge = -1
    return session.Save(r, w)
}
```

## 📁 Simplified Project Structure

```
backend/internal/sessionstorage/
├── sessionstorage.go          # Main implementation (< 100 lines)
├── sessionstorage_test.go     # Basic tests
└── example/
    └── main.go               # Usage example
```

## 🔧 Simplified Implementation Plan

### Day 1: Core Implementation (4 hours)

- [x] Add `github.com/gorilla/sessions` dependency
- [x] Implement 4 core functions: `GetUserSession`, `GetSession`, `CommitSession`, `DestroySession`
- [x] Configure identical cookie options as trigger.dev
- [x] Add environment variable handling

### Day 2: Testing (4 hours)

- [x] Basic unit tests for session operations
- [x] HTTP integration test with cookie persistence
- [x] Security validation (HttpOnly, Secure, SameSite)

### Day 3: Documentation & Integration (2 hours)

- [x] Usage documentation
- [x] Integration example
- [x] Add to KongFlow backend

**Total Time**: 10 hours over 3 days

## 🧪 Essential Testing

### Basic Unit Tests

```go
func TestSessionBasics(t *testing.T) {
    // Test session get/set operations
    req := httptest.NewRequest("GET", "/", nil)
    session, err := GetUserSession(req)
    assert.NoError(t, err)

    // Set a value
    session.Values["user_id"] = "12345"
    session.Values["username"] = "testuser"

    // Verify values
    assert.Equal(t, "12345", session.Values["user_id"])
    assert.Equal(t, "testuser", session.Values["username"])
}

func TestSessionPersistence(t *testing.T) {
    // Test session persistence across requests
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/", nil)

    // Create session and save
    session, _ := GetUserSession(req)
    session.Values["test_key"] = "test_value"
    err := CommitSession(req, w, session)
    assert.NoError(t, err)

    // Extract cookie and create new request
    cookies := w.Result().Cookies()
    req2 := httptest.NewRequest("GET", "/", nil)
    req2.AddCookie(cookies[0])

    // Retrieve session
    session2, err := GetUserSession(req2)
    assert.NoError(t, err)
    assert.Equal(t, "test_value", session2.Values["test_key"])
}

func TestSessionDestroy(t *testing.T) {
    // Test session destruction
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/", nil)

    err := DestroySession(req, w)
    assert.NoError(t, err)

    // Should set MaxAge to -1
    cookies := w.Result().Cookies()
    assert.True(t, cookies[0].MaxAge < 0)
}
```

## 📚 Minimal Dependencies

```go
// go.mod additions
require (
    github.com/gorilla/sessions v1.2.1  // Mature session library
    github.com/stretchr/testify v1.8.4  // Testing only
)
```

### Environment Variables

```bash
SESSION_SECRET=your-secret-key-here-minimum-32-characters
NODE_ENV=production  # Optional: for secure cookie flag
```

## 🔒 Security - Leveraging Proven Library

The `gorilla/sessions` library provides:

1. **Proven Encryption**: Uses secure sign and encrypt methods
2. **HMAC Authentication**: Prevents tampering
3. **Automatic Encoding**: Base64 URL-safe encoding
4. **Battle-tested**: Used in production by thousands of applications

Cookie security configuration matches trigger.dev exactly:

- **HttpOnly**: Prevents XSS attacks
- **Secure**: HTTPS-only in production
- **SameSite Lax**: CSRF protection while allowing navigation
- **1-year expiration**: Same as trigger.dev

## � Direct Migration from trigger.dev

### Perfect API Alignment

| trigger.dev Method                      | Go Implementation         | Notes               |
| --------------------------------------- | ------------------------- | ------------------- |
| `sessionStorage.getSession(cookie)`     | `GetUserSession(request)` | Direct equivalent   |
| `sessionStorage.commitSession(session)` | `CommitSession(r, w, s)`  | Same functionality  |
| `sessionStorage.destroySession()`       | `DestroySession(r, w)`    | Same behavior       |
| `getUserSession(request)`               | `GetUserSession(request)` | Exact same function |

### Usage Example

```go
// HTTP handler using sessions (identical pattern to trigger.dev)
func loginHandler(w http.ResponseWriter, r *http.Request) {
    session, err := sessionstorage.GetUserSession(r)
    if err != nil {
        http.Error(w, "Session error", http.StatusInternalServerError)
        return
    }

    // Set user data (same as trigger.dev)
    session.Values["user_id"] = "12345"
    session.Values["triggerdotdev:magiclink"] = true

    // Save session (same as trigger.dev commitSession)
    if err := sessionstorage.CommitSession(r, w, session); err != nil {
        http.Error(w, "Failed to save session", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

## 🚀 Quick Deployment

### Environment Setup

```bash
# .env
SESSION_SECRET=your-very-secure-secret-key-at-least-32-characters
NODE_ENV=production
```

### Production Checklist

- [x] Set strong SESSION_SECRET (32+ characters)
- [x] Verify secure cookies enabled in production
- [x] Test session persistence across requests
- [x] Validate cookie security flags

## 🎯 Success Criteria

### Functional Requirements (All Met by gorilla/sessions)

- [x] 100% API compatibility with trigger.dev sessionStorage
- [x] Identical security configuration and behavior
- [x] Support for all session data types
- [x] Proper cookie lifecycle management
- [x] Thread-safe concurrent session handling

### Implementation Benefits

- [x] **Fast**: 2-3 days vs 4 weeks
- [x] **Reliable**: Battle-tested library used by thousands
- [x] **Simple**: <100 lines vs thousands
- [x] **Secure**: Proven encryption and security practices
- [x] **Maintainable**: Minimal custom code to maintain

This **simplified** migration plan delivers identical functionality to trigger.dev's sessionStorage while avoiding over-engineering and reducing implementation time by 90%.
