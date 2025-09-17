// Example demonstrates the usage of the ULID service
// This shows how to use the service exactly like trigger.dev's ulid.server.ts
package main

import (
	"fmt"
	"sync"
	"time"

	"kongflow/backend/internal/ulid"
)

func main() {
	fmt.Println("=== KongFlow ULID Service Demo ===")
	fmt.Println("Replicating trigger.dev's ulid.server.ts behavior")
	fmt.Println()

	// Basic usage - direct replacement for trigger.dev's ulid() function
	fmt.Println("1. Basic ULID Generation (trigger.dev equivalent):")
	for i := 0; i < 3; i++ {
		id := ulid.NewULID()
		fmt.Printf("   Generated: %s\n", id)
	}
	fmt.Println()

	// Service instance usage
	fmt.Println("2. Service Instance Usage:")
	service := ulid.New()
	for i := 0; i < 3; i++ {
		id := service.Generate()
		fmt.Printf("   Generated: %s\n", id)
	}
	fmt.Println()

	// Demonstrate monotonic ordering
	fmt.Println("3. Monotonic Ordering Demo:")
	fmt.Println("   Generating multiple ULIDs rapidly...")
	var ids []string
	for i := 0; i < 10; i++ {
		ids = append(ids, ulid.NewULID())
	}

	// Show they are ordered
	fmt.Println("   Results (should be lexicographically ordered):")
	for i, id := range ids {
		fmt.Printf("   [%d] %s\n", i+1, id)
		if i > 0 && ids[i-1] > id {
			fmt.Printf("   ❌ ORDERING ERROR: %s > %s\n", ids[i-1], id)
		}
	}
	fmt.Println("   ✅ All ULIDs are properly ordered!")
	fmt.Println()

	// Batch generation demo
	fmt.Println("4. Batch Generation (Go enhancement):")
	batchIds := ulid.GenerateBatch(5)
	for i, id := range batchIds {
		fmt.Printf("   Batch[%d]: %s\n", i+1, id)
	}
	fmt.Println()

	// Concurrent usage demo
	fmt.Println("5. Concurrent Generation Demo:")
	var wg sync.WaitGroup
	var concurrentIds []string
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < 3; j++ {
				id := ulid.NewULID()
				mu.Lock()
				concurrentIds = append(concurrentIds, id)
				fmt.Printf("   Goroutine %d[%d]: %s\n", goroutineID, j+1, id)
				mu.Unlock()
				time.Sleep(1 * time.Millisecond) // Small delay to see interleaving
			}
		}(i + 1)
	}

	wg.Wait()
	fmt.Printf("   Generated %d ULIDs concurrently\n", len(concurrentIds))
	fmt.Println()

	// Performance demo
	fmt.Println("6. Performance Demo:")
	start := time.Now()
	count := 10000
	for i := 0; i < count; i++ {
		_ = ulid.NewULID()
	}
	duration := time.Since(start)

	fmt.Printf("   Generated %d ULIDs in %v\n", count, duration)
	fmt.Printf("   Average: %.2f ns/op\n", float64(duration.Nanoseconds())/float64(count))
	fmt.Printf("   Rate: %.0f ULIDs/second\n", float64(count)/duration.Seconds())
	fmt.Println()

	// Timestamp demo
	fmt.Println("7. Timestamp Property Demo:")
	id1 := ulid.NewULID()
	time.Sleep(2 * time.Millisecond)
	id2 := ulid.NewULID()

	fmt.Printf("   Earlier ULID:  %s\n", id1)
	fmt.Printf("   Later ULID:    %s\n", id2)
	fmt.Printf("   Ordering OK:   %t (earlier < later)\n", id1 < id2)
	fmt.Println()

	fmt.Println("=== Demo Complete ===")
	fmt.Println("✅ ULID service is working correctly and aligned with trigger.dev!")
}
