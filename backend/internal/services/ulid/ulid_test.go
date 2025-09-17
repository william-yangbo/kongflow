package ulid

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 80/20原则：专注于最重要的测试用例

// TestULIDFormat 验证ULID格式（最基础的20%测试覆盖80%问题）
func TestULIDFormat(t *testing.T) {
	tests := []struct {
		name     string
		generate func() string
	}{
		{"package function", NewULID},
		{"service instance", func() string { return New().Generate() }},
	}

	// ULID格式：26字符，base32编码，小写
	ulidRegex := regexp.MustCompile(`^[0-9a-hjkmnp-tv-z]{26}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.generate()

			// 验证长度
			assert.Len(t, id, 26, "ULID should be exactly 26 characters")

			// 验证格式（base32，小写）
			assert.True(t, ulidRegex.MatchString(id), "ULID should match base32 lowercase format")

			// 验证小写（与trigger.dev对齐）
			assert.Equal(t, id, id, "ULID should be lowercase")
			for _, char := range id {
				assert.False(t, char >= 'A' && char <= 'Z', "ULID should not contain uppercase characters")
			}
		})
	}
}

// TestUniqueness 验证唯一性（核心功能）
func TestUniqueness(t *testing.T) {
	const iterations = 1000
	ids := make(map[string]bool, iterations)

	service := New()

	for i := 0; i < iterations; i++ {
		id := service.Generate()
		assert.False(t, ids[id], "Generated ULID should be unique: %s", id)
		ids[id] = true
	}

	assert.Len(t, ids, iterations, "All generated ULIDs should be unique")
}

// TestMonotonicOrdering 验证单调性（与trigger.dev对齐的关键特性）
func TestMonotonicOrdering(t *testing.T) {
	service := New()

	// 在同一毫秒内生成多个ULID
	var ids []string
	start := time.Now()

	// 快速生成以确保在同一毫秒内
	for i := 0; i < 100; i++ {
		ids = append(ids, service.Generate())
	}

	duration := time.Since(start)
	t.Logf("Generated 100 ULIDs in %v", duration)

	// 验证字典序
	for i := 1; i < len(ids); i++ {
		assert.True(t, ids[i-1] <= ids[i],
			"ULIDs should be lexicographically ordered: %s should be <= %s",
			ids[i-1], ids[i])
	}
}

// TestConcurrentGeneration 验证线程安全（Go特有需求）
func TestConcurrentGeneration(t *testing.T) {
	const numGoroutines = 50
	const idsPerGoroutine = 20

	service := New()
	var wg sync.WaitGroup
	var mu sync.Mutex
	ids := make(map[string]bool, numGoroutines*idsPerGoroutine)

	// 并发生成
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < idsPerGoroutine; j++ {
				id := service.Generate()

				// 验证格式
				require.Len(t, id, 26)

				// 确保唯一性
				mu.Lock()
				require.False(t, ids[id], "Concurrent ULID should be unique: %s", id)
				ids[id] = true
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Len(t, ids, numGoroutines*idsPerGoroutine,
		"All concurrently generated ULIDs should be unique")
}

// TestBatchGeneration 验证批量生成功能
func TestBatchGeneration(t *testing.T) {
	service := New()

	t.Run("valid batch", func(t *testing.T) {
		ids := service.GenerateBatch(10)
		require.Len(t, ids, 10)

		// 验证每个ID的格式
		for i, id := range ids {
			assert.Len(t, id, 26, "Batch ULID %d should be 26 characters", i)
		}

		// 验证唯一性
		uniqueIds := make(map[string]bool)
		for _, id := range ids {
			assert.False(t, uniqueIds[id], "Batch ULIDs should be unique")
			uniqueIds[id] = true
		}

		// 验证单调性
		for i := 1; i < len(ids); i++ {
			assert.True(t, ids[i-1] <= ids[i],
				"Batch ULIDs should be ordered: %s <= %s", ids[i-1], ids[i])
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		assert.Nil(t, service.GenerateBatch(0), "Zero count should return nil")
		assert.Nil(t, service.GenerateBatch(-1), "Negative count should return nil")

		ids := service.GenerateBatch(1)
		assert.Len(t, ids, 1, "Single batch should return one ULID")
	})
}

// TestPackageLevelFunctions 验证包级别便利函数
func TestPackageLevelFunctions(t *testing.T) {
	t.Run("NewULID", func(t *testing.T) {
		id := NewULID()
		assert.Len(t, id, 26)

		// 验证与trigger.dev函数对等
		id2 := NewULID()
		assert.NotEqual(t, id, id2, "Package function should generate unique ULIDs")
	})

	t.Run("GenerateBatch", func(t *testing.T) {
		ids := GenerateBatch(5)
		assert.Len(t, ids, 5)

		for _, id := range ids {
			assert.Len(t, id, 26)
		}
	})
}

// TestTimestampProperty 验证时间戳属性（可排序性的基础）
func TestTimestampProperty(t *testing.T) {
	service := New()

	// 生成带时间间隔的ULID
	id1 := service.Generate()
	time.Sleep(1 * time.Millisecond) // 确保时间戳不同
	id2 := service.Generate()

	// 验证时间顺序反映在字典序中
	assert.True(t, id1 < id2,
		"ULIDs generated later should be lexicographically greater: %s < %s", id1, id2)

	// 验证时间戳部分（前10个字符）
	assert.True(t, id1[:10] <= id2[:10],
		"Timestamp portions should be ordered: %s <= %s", id1[:10], id2[:10])
}

// Benchmark tests for performance verification
func BenchmarkGenerate(b *testing.B) {
	service := New()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = service.Generate()
	}
}

func BenchmarkNewULID(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewULID()
	}
}

func BenchmarkConcurrentGenerate(b *testing.B) {
	service := New()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = service.Generate()
		}
	})
}

func BenchmarkBatchGenerate(b *testing.B) {
	service := New()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = service.GenerateBatch(10)
	}
}
