package secretstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSecretNotFound = errors.New("secret not found")

type Repository interface {
	GetSecret(ctx context.Context, key string) (*SecretStore, error)
	UpsertSecret(ctx context.Context, key string, value []byte) error
	DeleteSecret(ctx context.Context, key string) error
	ListSecretKeys(ctx context.Context) ([]ListSecretStoreKeysRow, error)
	GetSecretCount(ctx context.Context) (int64, error)
}

type repository struct {
	queries *Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		queries: New(db),
	}
}

func (r *repository) GetSecret(ctx context.Context, key string) (*SecretStore, error) {
	secret, err := r.queries.GetSecretStore(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}
	return &secret, nil
}

func (r *repository) UpsertSecret(ctx context.Context, key string, value []byte) error {
	return r.queries.UpsertSecretStore(ctx, UpsertSecretStoreParams{
		Key:   key,
		Value: value,
	})
}

func (r *repository) DeleteSecret(ctx context.Context, key string) error {
	return r.queries.DeleteSecretStore(ctx, key)
}

func (r *repository) ListSecretKeys(ctx context.Context) ([]ListSecretStoreKeysRow, error) {
	return r.queries.ListSecretStoreKeys(ctx)
}

func (r *repository) GetSecretCount(ctx context.Context) (int64, error) {
	return r.queries.GetSecretStoreCount(ctx)
}
