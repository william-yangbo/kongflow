package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestEventMatcher_BasicMatching(t *testing.T) {
	// 创建测试事件记录
	eventPayload := map[string]interface{}{
		"user_id": "123",
		"action":  "login",
	}

	payloadBytes, _ := json.Marshal(eventPayload)
	eventRecord := EventRecords{
		ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		EventID:   uuid.New().String(),
		Name:      "test.event",
		Source:    "test",
		Payload:   payloadBytes,
		Timestamp: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// 测试空过滤器（应该匹配所有事件）
	matcher := NewEventMatcher(eventRecord)
	emptyFilter := EventFilter{}
	assert.True(t, matcher.Matches(emptyFilter), "Empty filter should match everything")

	// 测试简单的有效负载匹配
	payloadFilter := EventFilter{
		Payload: map[string]interface{}{
			"action": "login",
		},
	}
	assert.True(t, matcher.Matches(payloadFilter), "Should match when payload contains filter criteria")
}

func TestUUIDConversionHelpers(t *testing.T) {
	originalUUID := uuid.New()

	// 测试 UUID 到 pgtype.UUID 的转换
	pgUUID := uuidToPgUUID(originalUUID)
	assert.True(t, pgUUID.Valid, "pgtype.UUID should be valid")
	assert.Equal(t, originalUUID, uuid.UUID(pgUUID.Bytes), "UUID conversion should preserve value")

	// 测试字符串到 pgtype.UUID 的转换
	uuidString := originalUUID.String()
	pgUUID2, err := stringToPgUUID(uuidString)
	assert.NoError(t, err, "String to pgtype.UUID conversion should not error")
	assert.True(t, pgUUID2.Valid, "Converted pgtype.UUID should be valid")
	assert.Equal(t, originalUUID, uuid.UUID(pgUUID2.Bytes), "String UUID conversion should preserve value")

	// 测试无效字符串
	_, err = stringToPgUUID("invalid-uuid")
	assert.Error(t, err, "Invalid UUID string should return error")
}

func TestUUIDHelperFunctions(t *testing.T) {
	// 测试UUID字符串验证
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	invalidUUID := "invalid-uuid"

	// 测试有效UUID字符串
	pgUUID, err := stringToPgUUID(validUUID)
	assert.NoError(t, err, "Valid UUID string should not return error")
	assert.True(t, pgUUID.Valid, "Converted pgtype.UUID should be valid")

	// 测试无效UUID字符串
	_, err = stringToPgUUID(invalidUUID)
	assert.Error(t, err, "Invalid UUID string should return error")
}
