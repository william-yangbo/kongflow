package endpoints

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEndpointNotFound      = errors.New("endpoint not found")
	ErrEndpointIndexNotFound = errors.New("endpoint index not found")
)

// Repository 数据访问接口
type Repository interface {
	// Endpoint CRUD
	CreateEndpoint(ctx context.Context, params CreateEndpointParams) (*CreateEndpointRow, error)
	GetEndpointByID(ctx context.Context, id uuid.UUID) (*GetEndpointByIDRow, error)
	GetEndpointBySlug(ctx context.Context, environmentID uuid.UUID, slug string) (*GetEndpointBySlugRow, error)
	UpdateEndpointURL(ctx context.Context, id uuid.UUID, url string) (*UpdateEndpointURLRow, error)
	UpsertEndpoint(ctx context.Context, params UpsertEndpointParams) (*UpsertEndpointRow, error)
	DeleteEndpoint(ctx context.Context, id uuid.UUID) error

	// EndpointIndex CRUD
	CreateEndpointIndex(ctx context.Context, params CreateEndpointIndexParams) (*CreateEndpointIndexRow, error)
	GetEndpointIndexByID(ctx context.Context, id uuid.UUID) (*GetEndpointIndexByIDRow, error)
	ListEndpointIndexes(ctx context.Context, endpointID uuid.UUID) ([]ListEndpointIndexesRow, error)
	DeleteEndpointIndex(ctx context.Context, id uuid.UUID) error

	// 事务支持
	WithTx(ctx context.Context, fn func(Repository) error) error
}

// repository 实现
type repository struct {
	queries *Queries
	db      *pgxpool.Pool
}

// NewRepository 创建仓库实例
func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		queries: New(db),
		db:      db,
	}
}

// WithTx 事务执行
func (r *repository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		}
	}()

	txRepo := &repository{
		queries: New(tx),
		db:      r.db,
	}

	err = fn(txRepo)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return err // 返回原始错误
		}
		return err
	}

	return tx.Commit(ctx)
}

// uuidToPgtype 将 Go UUID 转换为 pgtype.UUID
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// pgtypeToUUID 将 pgtype.UUID 转换为 Go UUID
func pgtypeToUUID(id pgtype.UUID) uuid.UUID {
	return id.Bytes
}

// CreateEndpoint 创建端点
func (r *repository) CreateEndpoint(ctx context.Context, params CreateEndpointParams) (*CreateEndpointRow, error) {
	endpoint, err := r.queries.CreateEndpoint(ctx, params)
	if err != nil {
		return nil, err
	}
	return &endpoint, nil
}

// GetEndpointByID 根据ID获取端点
func (r *repository) GetEndpointByID(ctx context.Context, id uuid.UUID) (*GetEndpointByIDRow, error) {
	endpoint, err := r.queries.GetEndpointByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEndpointNotFound
		}
		return nil, err
	}
	return &endpoint, nil
}

// GetEndpointBySlug 根据环境ID和slug获取端点
func (r *repository) GetEndpointBySlug(ctx context.Context, environmentID uuid.UUID, slug string) (*GetEndpointBySlugRow, error) {
	endpoint, err := r.queries.GetEndpointBySlug(ctx, GetEndpointBySlugParams{
		EnvironmentID: uuidToPgtype(environmentID),
		Slug:          slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEndpointNotFound
		}
		return nil, err
	}
	return &endpoint, nil
}

// UpdateEndpointURL 更新端点URL
func (r *repository) UpdateEndpointURL(ctx context.Context, id uuid.UUID, url string) (*UpdateEndpointURLRow, error) {
	endpoint, err := r.queries.UpdateEndpointURL(ctx, UpdateEndpointURLParams{
		ID:  uuidToPgtype(id),
		Url: url,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEndpointNotFound
		}
		return nil, err
	}
	return &endpoint, nil
}

// UpsertEndpoint 创建或更新端点
func (r *repository) UpsertEndpoint(ctx context.Context, params UpsertEndpointParams) (*UpsertEndpointRow, error) {
	endpoint, err := r.queries.UpsertEndpoint(ctx, params)
	if err != nil {
		return nil, err
	}
	return &endpoint, nil
}

// DeleteEndpoint 删除端点
func (r *repository) DeleteEndpoint(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteEndpoint(ctx, uuidToPgtype(id))
}

// CreateEndpointIndex 创建端点索引
func (r *repository) CreateEndpointIndex(ctx context.Context, params CreateEndpointIndexParams) (*CreateEndpointIndexRow, error) {
	index, err := r.queries.CreateEndpointIndex(ctx, params)
	if err != nil {
		return nil, err
	}
	return &index, nil
}

// GetEndpointIndexByID 根据ID获取端点索引
func (r *repository) GetEndpointIndexByID(ctx context.Context, id uuid.UUID) (*GetEndpointIndexByIDRow, error) {
	index, err := r.queries.GetEndpointIndexByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEndpointIndexNotFound
		}
		return nil, err
	}
	return &index, nil
}

// ListEndpointIndexes 列出端点的所有索引
func (r *repository) ListEndpointIndexes(ctx context.Context, endpointID uuid.UUID) ([]ListEndpointIndexesRow, error) {
	return r.queries.ListEndpointIndexes(ctx, uuidToPgtype(endpointID))
}

// DeleteEndpointIndex 删除端点索引
func (r *repository) DeleteEndpointIndex(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteEndpointIndex(ctx, uuidToPgtype(id))
}
