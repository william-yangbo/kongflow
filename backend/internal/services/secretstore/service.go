package secretstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetSecret 获取密钥并反序列化到 target
func (s *Service) GetSecret(ctx context.Context, key string, target interface{}) error {
	secret, err := s.repo.GetSecret(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	if err := json.Unmarshal(secret.Value, target); err != nil {
		return fmt.Errorf("failed to unmarshal secret %s: %w", key, err)
	}

	return nil
}

// SetSecret 序列化 value 并存储
func (s *Service) SetSecret(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	if err := s.repo.UpsertSecret(ctx, key, data); err != nil {
		return fmt.Errorf("failed to set secret %s: %w", key, err)
	}

	return nil
}

// GetSecretOrThrow 如果不存在则返回错误 (兼容 trigger.dev 接口)
func (s *Service) GetSecretOrThrow(ctx context.Context, key string, target interface{}) error {
	if err := s.GetSecret(ctx, key, target); err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return fmt.Errorf("secret %s not found", key)
		}
		return err
	}
	return nil
}
