package events

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository Events 数据仓储接口，遵循已迁移服务的模式
type Repository interface {
	// EventRecord 操作
	CreateEventRecord(ctx context.Context, params CreateEventRecordParams) (EventRecords, error)
	GetEventRecordByID(ctx context.Context, id pgtype.UUID) (EventRecords, error)
	GetEventRecordByEventID(ctx context.Context, params GetEventRecordByEventIDParams) (EventRecords, error)
	UpdateEventRecordDeliveredAt(ctx context.Context, params UpdateEventRecordDeliveredAtParams) error
	ListEventRecords(ctx context.Context, params ListEventRecordsParams) ([]EventRecords, error)
	CountEventRecords(ctx context.Context, params CountEventRecordsParams) (int64, error)
	ListPendingEventRecords(ctx context.Context, params ListPendingEventRecordsParams) ([]EventRecords, error)

	// EventDispatcher 操作
	GetEventDispatcherByID(ctx context.Context, id pgtype.UUID) (EventDispatchers, error)
	FindEventDispatchers(ctx context.Context, params FindEventDispatchersParams) ([]EventDispatchers, error)
	ListEventDispatchers(ctx context.Context, params ListEventDispatchersParams) ([]EventDispatchers, error)
	CountEventDispatchers(ctx context.Context, params CountEventDispatchersParams) (int64, error)
	CreateEventDispatcher(ctx context.Context, params CreateEventDispatcherParams) (EventDispatchers, error)
	UpsertEventDispatcher(ctx context.Context, params UpsertEventDispatcherParams) (EventDispatchers, error)
	UpdateEventDispatcherEnabled(ctx context.Context, params UpdateEventDispatcherEnabledParams) error
	DeleteEventDispatcher(ctx context.Context, id pgtype.UUID) error

	// 事务支持
	WithTx(ctx context.Context, fn func(Repository) error) error
	WithTxAndReturn(ctx context.Context, fn func(Repository, pgx.Tx) error) error
}

// repository 实现
type repository struct {
	queries Querier
	db      *pgxpool.Pool
}

// NewRepository 创建仓储实例
func NewRepository(queries Querier, db *pgxpool.Pool) Repository {
	return &repository{
		queries: queries,
		db:      db,
	}
}

// EventRecord 操作实现
func (r *repository) CreateEventRecord(ctx context.Context, params CreateEventRecordParams) (EventRecords, error) {
	return r.queries.CreateEventRecord(ctx, params)
}

func (r *repository) GetEventRecordByID(ctx context.Context, id pgtype.UUID) (EventRecords, error) {
	return r.queries.GetEventRecordByID(ctx, id)
}

func (r *repository) GetEventRecordByEventID(ctx context.Context, params GetEventRecordByEventIDParams) (EventRecords, error) {
	return r.queries.GetEventRecordByEventID(ctx, params)
}

func (r *repository) UpdateEventRecordDeliveredAt(ctx context.Context, params UpdateEventRecordDeliveredAtParams) error {
	return r.queries.UpdateEventRecordDeliveredAt(ctx, params)
}

func (r *repository) ListEventRecords(ctx context.Context, params ListEventRecordsParams) ([]EventRecords, error) {
	return r.queries.ListEventRecords(ctx, params)
}

func (r *repository) CountEventRecords(ctx context.Context, params CountEventRecordsParams) (int64, error) {
	return r.queries.CountEventRecords(ctx, params)
}

func (r *repository) ListPendingEventRecords(ctx context.Context, params ListPendingEventRecordsParams) ([]EventRecords, error) {
	return r.queries.ListPendingEventRecords(ctx, params)
}

// EventDispatcher 操作实现
func (r *repository) GetEventDispatcherByID(ctx context.Context, id pgtype.UUID) (EventDispatchers, error) {
	return r.queries.GetEventDispatcherByID(ctx, id)
}

func (r *repository) FindEventDispatchers(ctx context.Context, params FindEventDispatchersParams) ([]EventDispatchers, error) {
	return r.queries.FindEventDispatchers(ctx, params)
}

func (r *repository) ListEventDispatchers(ctx context.Context, params ListEventDispatchersParams) ([]EventDispatchers, error) {
	return r.queries.ListEventDispatchers(ctx, params)
}

func (r *repository) CountEventDispatchers(ctx context.Context, params CountEventDispatchersParams) (int64, error) {
	return r.queries.CountEventDispatchers(ctx, params)
}

func (r *repository) CreateEventDispatcher(ctx context.Context, params CreateEventDispatcherParams) (EventDispatchers, error) {
	return r.queries.CreateEventDispatcher(ctx, params)
}

func (r *repository) UpsertEventDispatcher(ctx context.Context, params UpsertEventDispatcherParams) (EventDispatchers, error) {
	return r.queries.UpsertEventDispatcher(ctx, params)
}

func (r *repository) UpdateEventDispatcherEnabled(ctx context.Context, params UpdateEventDispatcherEnabledParams) error {
	return r.queries.UpdateEventDispatcherEnabled(ctx, params)
}

func (r *repository) DeleteEventDispatcher(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteEventDispatcher(ctx, id)
}

// WithTx 事务支持
func (r *repository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 创建事务查询器
	txQueries := New(tx)
	txRepo := &repository{
		queries: txQueries,
		db:      r.db,
	}

	// 执行事务内的操作
	if err := fn(txRepo); err != nil {
		return err
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTxAndReturn 事务支持（带事务对象返回）
func (r *repository) WithTxAndReturn(ctx context.Context, fn func(Repository, pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 创建事务查询器
	txQueries := New(tx)
	txRepo := &repository{
		queries: txQueries,
		db:      r.db,
	}

	// 执行事务内的操作
	if err := fn(txRepo, tx); err != nil {
		return err
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
