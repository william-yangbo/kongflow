// Package ulid provides ULID (Universally Unique Lexicographically Sortable Identifier) generation
// services with strict alignment to trigger.dev's implementation while adapting Go best practices.
//
// This package implements the exact behavior of trigger.dev's ulid.server.ts:
// - Monotonic factory pattern for consistent ordering
// - Lowercase output format
// - Thread-safe generation for concurrent use
package ulid

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service provides thread-safe ULID generation with monotonic ordering.
// It replicates the behavior of trigger.dev's monotonicFactory pattern.
type Service struct {
	entropy *ulid.MonotonicEntropy
	mu      sync.Mutex
}

// New creates a new ULID service instance with monotonic entropy source.
// This ensures monotonic ordering within the same millisecond for this instance,
// exactly matching trigger.dev's monotonicFactory behavior.
func New() *Service {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	return &Service{
		entropy: entropy,
	}
}

// Generate creates a new ULID string with lowercase output.
// This method is thread-safe and maintains monotonic ordering.
//
// Returns a 26-character ULID string in lowercase format, exactly
// matching trigger.dev's ulid() function behavior.
func (s *Service) Generate() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ULID with current timestamp and monotonic entropy
	id := ulid.MustNew(ulid.Timestamp(time.Now()), s.entropy)

	// Convert to lowercase to match trigger.dev behavior
	return strings.ToLower(id.String())
}

// GenerateBatch creates multiple ULIDs in a single operation for efficiency.
// This is a Go-specific enhancement while maintaining trigger.dev compatibility.
//
// All ULIDs in the batch maintain monotonic ordering and use lowercase format.
func (s *Service) GenerateBatch(count int) []string {
	if count <= 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]string, count)

	for i := 0; i < count; i++ {
		// Each call to monotonic entropy will ensure proper ordering
		id := ulid.MustNew(ulid.Timestamp(time.Now()), s.entropy)
		results[i] = strings.ToLower(id.String())
	}

	return results
}

// Package-level service instance for convenience (singleton pattern)
var defaultService = New()

// NewULID generates a new ULID using the default service instance.
// This function provides a direct replacement for trigger.dev's ulid() function.
//
// Example usage:
//
//	id := ulid.NewULID()
//	fmt.Println(id) // "01h4pg5qr7kjb9s8vw9x1234mt"
func NewULID() string {
	return defaultService.Generate()
}

// GenerateBatch generates multiple ULIDs using the default service instance.
// This is a convenience function for batch operations.
func GenerateBatch(count int) []string {
	return defaultService.GenerateBatch(count)
}
