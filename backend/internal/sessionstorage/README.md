# SessionStorage Service

A cookie-based session management service that strictly aligns with trigger.dev's sessionStorage implementation.

## Overview

This package provides cookie-based session storage with the exact same configuration and API as trigger.dev:

- **Cookie Configuration**: Identical security settings (HttpOnly, Secure, SameSite)
- **API Compatibility**: Same method signatures and behavior
- **Security**: Secure session handling with environment-based secrets
- **Thread-Safe**: Concurrent access support with proper synchronization

## Quick Start

### Environment Setup

```bash
export SESSION_SECRET="your-secure-session-secret-key"
```

### Basic Usage

```go
package main

import (
    "net/http"
    "kongflow/backend/internal/sessionstorage"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
    // Get user session
    session, err := sessionstorage.GetUserSession(r)
    if err != nil {
        http.Error(w, "Session error", http.StatusInternalServerError)
        return
    }

    // Store user data
    session.Values["userID"] = "user-123"
    session.Values["username"] = "john_doe"

    // Commit session
    if err := sessionstorage.CommitSession(r, w, session); err != nil {
        http.Error(w, "Failed to save session", http.StatusInternalServerError)
        return
    }

    w.Write([]byte("Login successful"))
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
    // Destroy session
    if err := sessionstorage.DestroySession(r, w); err != nil {
        http.Error(w, "Failed to destroy session", http.StatusInternalServerError)
        return
    }

    w.Write([]byte("Logout successful"))
}
```

## API Reference

### GetUserSession

```go
func GetUserSession(r *http.Request) (*sessions.Session, error)
```

Retrieves the user session from the request. Creates a new session if none exists.

**Parameters:**

- `r *http.Request`: The HTTP request

**Returns:**

- `*sessions.Session`: The session object
- `error`: Error if session retrieval fails

### CommitSession

```go
func CommitSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) error
```

Saves the session data to the response cookies.

**Parameters:**

- `r *http.Request`: The HTTP request
- `w http.ResponseWriter`: The HTTP response writer
- `session *sessions.Session`: The session to save

**Returns:**

- `error`: Error if session commit fails

### DestroySession

```go
func DestroySession(r *http.Request, w http.ResponseWriter) error
```

Destroys the user session by setting expiration in the past.

**Parameters:**

- `r *http.Request`: The HTTP request
- `w http.ResponseWriter`: The HTTP response writer

**Returns:**

- `error`: Error if session destruction fails

## Configuration

### Environment Variables

| Variable         | Description                       | Required |
| ---------------- | --------------------------------- | -------- |
| `SESSION_SECRET` | Secret key for session encryption | Yes      |

### Cookie Settings

The service uses the following cookie configuration to match trigger.dev:

```go
Options: &sessions.Options{
    Path:     "/",
    MaxAge:   365 * 24 * 60 * 60, // 1 year
    HttpOnly: true,
    Secure:   true,  // Auto-disabled in non-HTTPS environments
    SameSite: http.SameSiteLaxMode,
}
```

## Security Features

### Automatic HTTPS Detection

The service automatically adjusts cookie security based on the environment:

- **HTTPS**: Full security (Secure flag enabled)
- **HTTP**: Development mode (Secure flag disabled for local development)

### Session Encryption

All session data is encrypted using the `SESSION_SECRET` environment variable.

### Cookie Security

- **HttpOnly**: Prevents XSS attacks
- **Secure**: HTTPS-only transmission (when available)
- **SameSite Lax**: CSRF protection while maintaining usability

## Testing

Run the test suite:

```bash
SESSION_SECRET=test-secret go test ./internal/sessionstorage -v
```

### Test Coverage

- ✅ Basic session operations
- ✅ Session persistence across requests
- ✅ Session destruction
- ✅ Cookie security configuration
- ✅ Environment variable validation
- ✅ Concurrent access safety
- ✅ Custom session names
- ✅ Production vs development modes

## Migration from trigger.dev

This service is designed as a drop-in replacement for trigger.dev's sessionStorage:

### trigger.dev Code

```typescript
// trigger.dev sessionStorage
const session = await sessionStorage.getSession(request.headers.get('Cookie'));
session.set('userID', userID);
return json(data, {
  headers: {
    'Set-Cookie': await sessionStorage.commitSession(session),
  },
});
```

### KongFlow Equivalent

```go
// KongFlow sessionStorage
session, err := sessionstorage.GetUserSession(r)
session.Values["userID"] = userID
err = sessionstorage.CommitSession(r, w, session)
```

## Performance

- **Memory Efficient**: Minimal overhead using gorilla/sessions
- **Thread-Safe**: Concurrent request handling
- **Fast**: Direct cookie-based storage (no database queries)

## Dependencies

- [gorilla/sessions](https://github.com/gorilla/sessions) v1.4.0 - Secure cookie sessions
- [gorilla/securecookie](https://github.com/gorilla/securecookie) v1.1.2 - Cookie encryption

## License

This package is part of the KongFlow backend system.
