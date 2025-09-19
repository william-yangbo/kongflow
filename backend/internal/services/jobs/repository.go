package jobs

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository 作业仓储接口，遵循 endpoints 服务的模式
type Repository interface {
	// Job operations
	CreateJob(ctx context.Context, params CreateJobParams) (Jobs, error)
	GetJobByID(ctx context.Context, id pgtype.UUID) (Jobs, error)
	GetJobBySlug(ctx context.Context, projectID pgtype.UUID, slug string) (Jobs, error)
	UpsertJob(ctx context.Context, params UpsertJobParams) (Jobs, error)
	ListJobsByProject(ctx context.Context, params ListJobsByProjectParams) ([]Jobs, error)
	CountJobsByProject(ctx context.Context, projectID pgtype.UUID) (int64, error)
	UpdateJob(ctx context.Context, params UpdateJobParams) (Jobs, error)
	DeleteJob(ctx context.Context, id pgtype.UUID) error

	// JobVersion operations
	CreateJobVersion(ctx context.Context, params CreateJobVersionParams) (JobVersions, error)
	GetJobVersionByID(ctx context.Context, id pgtype.UUID) (JobVersions, error)
	GetJobVersionByJobAndVersion(ctx context.Context, params GetJobVersionByJobAndVersionParams) (JobVersions, error)
	UpsertJobVersion(ctx context.Context, params UpsertJobVersionParams) (JobVersions, error)
	ListJobVersionsByJob(ctx context.Context, params ListJobVersionsByJobParams) ([]JobVersions, error)
	GetLatestJobVersion(ctx context.Context, params GetLatestJobVersionParams) (JobVersions, error)
	CountLaterJobVersions(ctx context.Context, params CountLaterJobVersionsParams) (int64, error)
	UpdateJobVersionProperties(ctx context.Context, params UpdateJobVersionPropertiesParams) (JobVersions, error)
	DeleteJobVersion(ctx context.Context, id pgtype.UUID) error

	// JobQueue operations
	CreateJobQueue(ctx context.Context, params CreateJobQueueParams) (JobQueues, error)
	GetJobQueueByID(ctx context.Context, id pgtype.UUID) (JobQueues, error)
	GetJobQueueByName(ctx context.Context, params GetJobQueueByNameParams) (JobQueues, error)
	UpsertJobQueue(ctx context.Context, params UpsertJobQueueParams) (JobQueues, error)
	ListJobQueuesByEnvironment(ctx context.Context, params ListJobQueuesByEnvironmentParams) ([]JobQueues, error)
	UpdateJobQueueCounts(ctx context.Context, params UpdateJobQueueCountsParams) (JobQueues, error)
	IncrementJobCount(ctx context.Context, id pgtype.UUID) (JobQueues, error)
	DecrementJobCount(ctx context.Context, id pgtype.UUID) (JobQueues, error)
	DeleteJobQueue(ctx context.Context, id pgtype.UUID) error

	// JobAlias operations
	CreateJobAlias(ctx context.Context, params CreateJobAliasParams) (JobAliases, error)
	GetJobAliasByID(ctx context.Context, id pgtype.UUID) (JobAliases, error)
	GetJobAliasByName(ctx context.Context, params GetJobAliasByNameParams) (JobAliases, error)
	UpsertJobAlias(ctx context.Context, params UpsertJobAliasParams) (JobAliases, error)
	ListJobAliasesByJob(ctx context.Context, params ListJobAliasesByJobParams) ([]JobAliases, error)
	DeleteJobAlias(ctx context.Context, id pgtype.UUID) error
	DeleteJobAliasesByJob(ctx context.Context, jobID pgtype.UUID) error

	// EventExample operations
	CreateEventExample(ctx context.Context, params CreateEventExampleParams) (EventExamples, error)
	GetEventExampleByID(ctx context.Context, id pgtype.UUID) (EventExamples, error)
	GetEventExampleBySlug(ctx context.Context, params GetEventExampleBySlugParams) (EventExamples, error)
	UpsertEventExample(ctx context.Context, params UpsertEventExampleParams) (EventExamples, error)
	ListEventExamplesByJobVersion(ctx context.Context, jobVersionID pgtype.UUID) ([]EventExamples, error)
	DeleteEventExample(ctx context.Context, id pgtype.UUID) error
	DeleteEventExamplesByJobVersion(ctx context.Context, jobVersionID pgtype.UUID) error
	DeleteEventExamplesNotInList(ctx context.Context, params DeleteEventExamplesNotInListParams) error

	// EventRecord operations
	CreateEventRecord(ctx context.Context, params CreateEventRecordParams) (EventRecords, error)
	GetEventRecordByID(ctx context.Context, id pgtype.UUID) (EventRecords, error)
	GetEventRecordByEventID(ctx context.Context, params GetEventRecordByEventIDParams) (EventRecords, error)
	ListEventRecordsByEnvironment(ctx context.Context, params ListEventRecordsByEnvironmentParams) ([]EventRecords, error)
	ListTestEventRecords(ctx context.Context, params ListTestEventRecordsParams) ([]EventRecords, error)
	ListEventRecordsByNameAndSource(ctx context.Context, params ListEventRecordsByNameAndSourceParams) ([]EventRecords, error)
	CountEventRecordsByEnvironment(ctx context.Context, environmentID pgtype.UUID) (int64, error)
	CountTestEventRecords(ctx context.Context, environmentID pgtype.UUID) (int64, error)
	UpdateEventRecord(ctx context.Context, params UpdateEventRecordParams) (EventRecords, error)
	DeleteEventRecord(ctx context.Context, id pgtype.UUID) error
	DeleteEventRecordByEventID(ctx context.Context, params DeleteEventRecordByEventIDParams) error

	// Transaction support
	WithTx(ctx context.Context, fn func(Repository) error) error
}

// repository 仓储实现
type repository struct {
	db      *pgxpool.Pool
	queries *Queries
}

// NewRepository 创建新的仓储实例
func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db:      db,
		queries: New(db),
	}
}

// WithTx 在事务中执行操作
func (r *repository) WithTx(ctx context.Context, fn func(Repository) error) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txRepo := &repository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	return fn(txRepo)
}

// Job operations implementation
func (r *repository) CreateJob(ctx context.Context, params CreateJobParams) (Jobs, error) {
	return r.queries.CreateJob(ctx, params)
}

func (r *repository) GetJobByID(ctx context.Context, id pgtype.UUID) (Jobs, error) {
	return r.queries.GetJobByID(ctx, id)
}

func (r *repository) GetJobBySlug(ctx context.Context, projectID pgtype.UUID, slug string) (Jobs, error) {
	return r.queries.GetJobBySlug(ctx, GetJobBySlugParams{
		ProjectID: projectID,
		Slug:      slug,
	})
}

func (r *repository) UpsertJob(ctx context.Context, params UpsertJobParams) (Jobs, error) {
	return r.queries.UpsertJob(ctx, params)
}

func (r *repository) ListJobsByProject(ctx context.Context, params ListJobsByProjectParams) ([]Jobs, error) {
	return r.queries.ListJobsByProject(ctx, params)
}

func (r *repository) CountJobsByProject(ctx context.Context, projectID pgtype.UUID) (int64, error) {
	return r.queries.CountJobsByProject(ctx, projectID)
}

func (r *repository) UpdateJob(ctx context.Context, params UpdateJobParams) (Jobs, error) {
	return r.queries.UpdateJob(ctx, params)
}

func (r *repository) DeleteJob(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteJob(ctx, id)
}

// JobVersion operations implementation
func (r *repository) CreateJobVersion(ctx context.Context, params CreateJobVersionParams) (JobVersions, error) {
	return r.queries.CreateJobVersion(ctx, params)
}

func (r *repository) GetJobVersionByID(ctx context.Context, id pgtype.UUID) (JobVersions, error) {
	return r.queries.GetJobVersionByID(ctx, id)
}

func (r *repository) GetJobVersionByJobAndVersion(ctx context.Context, params GetJobVersionByJobAndVersionParams) (JobVersions, error) {
	return r.queries.GetJobVersionByJobAndVersion(ctx, params)
}

func (r *repository) UpsertJobVersion(ctx context.Context, params UpsertJobVersionParams) (JobVersions, error) {
	return r.queries.UpsertJobVersion(ctx, params)
}

func (r *repository) ListJobVersionsByJob(ctx context.Context, params ListJobVersionsByJobParams) ([]JobVersions, error) {
	return r.queries.ListJobVersionsByJob(ctx, params)
}

func (r *repository) GetLatestJobVersion(ctx context.Context, params GetLatestJobVersionParams) (JobVersions, error) {
	return r.queries.GetLatestJobVersion(ctx, params)
}

func (r *repository) CountLaterJobVersions(ctx context.Context, params CountLaterJobVersionsParams) (int64, error) {
	return r.queries.CountLaterJobVersions(ctx, params)
}

func (r *repository) UpdateJobVersionProperties(ctx context.Context, params UpdateJobVersionPropertiesParams) (JobVersions, error) {
	return r.queries.UpdateJobVersionProperties(ctx, params)
}

func (r *repository) DeleteJobVersion(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteJobVersion(ctx, id)
}

// JobQueue operations implementation
func (r *repository) CreateJobQueue(ctx context.Context, params CreateJobQueueParams) (JobQueues, error) {
	return r.queries.CreateJobQueue(ctx, params)
}

func (r *repository) GetJobQueueByID(ctx context.Context, id pgtype.UUID) (JobQueues, error) {
	return r.queries.GetJobQueueByID(ctx, id)
}

func (r *repository) GetJobQueueByName(ctx context.Context, params GetJobQueueByNameParams) (JobQueues, error) {
	return r.queries.GetJobQueueByName(ctx, params)
}

func (r *repository) UpsertJobQueue(ctx context.Context, params UpsertJobQueueParams) (JobQueues, error) {
	return r.queries.UpsertJobQueue(ctx, params)
}

func (r *repository) ListJobQueuesByEnvironment(ctx context.Context, params ListJobQueuesByEnvironmentParams) ([]JobQueues, error) {
	return r.queries.ListJobQueuesByEnvironment(ctx, params)
}

func (r *repository) UpdateJobQueueCounts(ctx context.Context, params UpdateJobQueueCountsParams) (JobQueues, error) {
	return r.queries.UpdateJobQueueCounts(ctx, params)
}

func (r *repository) IncrementJobCount(ctx context.Context, id pgtype.UUID) (JobQueues, error) {
	return r.queries.IncrementJobCount(ctx, id)
}

func (r *repository) DecrementJobCount(ctx context.Context, id pgtype.UUID) (JobQueues, error) {
	return r.queries.DecrementJobCount(ctx, id)
}

func (r *repository) DeleteJobQueue(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteJobQueue(ctx, id)
}

// JobAlias operations implementation
func (r *repository) CreateJobAlias(ctx context.Context, params CreateJobAliasParams) (JobAliases, error) {
	return r.queries.CreateJobAlias(ctx, params)
}

func (r *repository) GetJobAliasByID(ctx context.Context, id pgtype.UUID) (JobAliases, error) {
	return r.queries.GetJobAliasByID(ctx, id)
}

func (r *repository) GetJobAliasByName(ctx context.Context, params GetJobAliasByNameParams) (JobAliases, error) {
	return r.queries.GetJobAliasByName(ctx, params)
}

func (r *repository) UpsertJobAlias(ctx context.Context, params UpsertJobAliasParams) (JobAliases, error) {
	return r.queries.UpsertJobAlias(ctx, params)
}

func (r *repository) ListJobAliasesByJob(ctx context.Context, params ListJobAliasesByJobParams) ([]JobAliases, error) {
	return r.queries.ListJobAliasesByJob(ctx, params)
}

func (r *repository) DeleteJobAlias(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteJobAlias(ctx, id)
}

func (r *repository) DeleteJobAliasesByJob(ctx context.Context, jobID pgtype.UUID) error {
	return r.queries.DeleteJobAliasesByJob(ctx, jobID)
}

// EventExample operations implementation
func (r *repository) CreateEventExample(ctx context.Context, params CreateEventExampleParams) (EventExamples, error) {
	return r.queries.CreateEventExample(ctx, params)
}

func (r *repository) GetEventExampleByID(ctx context.Context, id pgtype.UUID) (EventExamples, error) {
	return r.queries.GetEventExampleByID(ctx, id)
}

func (r *repository) GetEventExampleBySlug(ctx context.Context, params GetEventExampleBySlugParams) (EventExamples, error) {
	return r.queries.GetEventExampleBySlug(ctx, params)
}

func (r *repository) UpsertEventExample(ctx context.Context, params UpsertEventExampleParams) (EventExamples, error) {
	return r.queries.UpsertEventExample(ctx, params)
}

func (r *repository) ListEventExamplesByJobVersion(ctx context.Context, jobVersionID pgtype.UUID) ([]EventExamples, error) {
	return r.queries.ListEventExamplesByJobVersion(ctx, jobVersionID)
}

func (r *repository) DeleteEventExample(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteEventExample(ctx, id)
}

func (r *repository) DeleteEventExamplesByJobVersion(ctx context.Context, jobVersionID pgtype.UUID) error {
	return r.queries.DeleteEventExamplesByJobVersion(ctx, jobVersionID)
}

func (r *repository) DeleteEventExamplesNotInList(ctx context.Context, params DeleteEventExamplesNotInListParams) error {
	return r.queries.DeleteEventExamplesNotInList(ctx, params)
}

// EventRecord repository implementations

func (r *repository) CreateEventRecord(ctx context.Context, params CreateEventRecordParams) (EventRecords, error) {
	return r.queries.CreateEventRecord(ctx, params)
}

func (r *repository) GetEventRecordByID(ctx context.Context, id pgtype.UUID) (EventRecords, error) {
	return r.queries.GetEventRecordByID(ctx, id)
}

func (r *repository) GetEventRecordByEventID(ctx context.Context, params GetEventRecordByEventIDParams) (EventRecords, error) {
	return r.queries.GetEventRecordByEventID(ctx, params)
}

func (r *repository) ListEventRecordsByEnvironment(ctx context.Context, params ListEventRecordsByEnvironmentParams) ([]EventRecords, error) {
	return r.queries.ListEventRecordsByEnvironment(ctx, params)
}

func (r *repository) ListTestEventRecords(ctx context.Context, params ListTestEventRecordsParams) ([]EventRecords, error) {
	return r.queries.ListTestEventRecords(ctx, params)
}

func (r *repository) ListEventRecordsByNameAndSource(ctx context.Context, params ListEventRecordsByNameAndSourceParams) ([]EventRecords, error) {
	return r.queries.ListEventRecordsByNameAndSource(ctx, params)
}

func (r *repository) CountEventRecordsByEnvironment(ctx context.Context, environmentID pgtype.UUID) (int64, error) {
	return r.queries.CountEventRecordsByEnvironment(ctx, environmentID)
}

func (r *repository) CountTestEventRecords(ctx context.Context, environmentID pgtype.UUID) (int64, error) {
	return r.queries.CountTestEventRecords(ctx, environmentID)
}

func (r *repository) UpdateEventRecord(ctx context.Context, params UpdateEventRecordParams) (EventRecords, error) {
	return r.queries.UpdateEventRecord(ctx, params)
}

func (r *repository) DeleteEventRecord(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteEventRecord(ctx, id)
}

func (r *repository) DeleteEventRecordByEventID(ctx context.Context, params DeleteEventRecordByEventIDParams) error {
	return r.queries.DeleteEventRecordByEventID(ctx, params)
}
