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
