package apiauth

import (
	"context"
	"fmt"

	"kongflow/backend/internal/shared"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// repository implements Repository interface combining shared + apiauth queries
type repository struct {
	sharedQueries  *shared.Queries // User, Organization, Project, RuntimeEnvironment queries
	apiAuthQueries *Queries        // PersonalAccessToken, OrganizationAccessToken queries
	db             DBTX
}

// NewRepository creates a new repository instance with hybrid architecture
func NewRepository(db DBTX) Repository {
	return &repository{
		sharedQueries:  shared.New(db),
		apiAuthQueries: New(db),
		db:             db,
	}
}

// FindEnvironmentByAPIKey finds runtime environment by API key using shared queries
func (r *repository) FindEnvironmentByAPIKey(ctx context.Context, apiKey string) (*RuntimeEnvironment, error) {
	env, err := r.sharedQueries.FindRuntimeEnvironmentByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("environment not found for API key: %w", err)
	}

	return &RuntimeEnvironment{
		ID:             env.ID,
		Slug:           env.Slug,
		APIKey:         env.ApiKey,
		Type:           EnvironmentType(env.Type),
		OrganizationID: env.OrganizationID,
		ProjectID:      env.ProjectID,
		OrgMemberID:    env.OrgMemberID,
		CreatedAt:      env.CreatedAt.Time,
		UpdatedAt:      env.UpdatedAt.Time,
	}, nil
}

// FindEnvironmentByPublicAPIKey finds runtime environment by public API key (non-production only)
func (r *repository) FindEnvironmentByPublicAPIKey(ctx context.Context, apiKey string, branch *string) (*RuntimeEnvironment, error) {
	env, err := r.sharedQueries.FindRuntimeEnvironmentByPublicAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("public environment not found for API key: %w", err)
	}

	return &RuntimeEnvironment{
		ID:             env.ID,
		Slug:           env.Slug,
		APIKey:         env.ApiKey,
		Type:           EnvironmentType(env.Type),
		OrganizationID: env.OrganizationID,
		ProjectID:      env.ProjectID,
		OrgMemberID:    env.OrgMemberID,
		CreatedAt:      env.CreatedAt.Time,
		UpdatedAt:      env.UpdatedAt.Time,
	}, nil
}

// GetEnvironmentWithProjectAndOrg gets environment with full project and organization context
func (r *repository) GetEnvironmentWithProjectAndOrg(ctx context.Context, envID string) (*AuthenticatedEnvironment, error) {
	envUUID := pgtype.UUID{}
	if err := envUUID.Scan(envID); err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	result, err := r.sharedQueries.GetEnvironmentWithProjectAndOrg(ctx, envUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment with context: %w", err)
	}

	return &AuthenticatedEnvironment{
		Environment: RuntimeEnvironment{
			ID:             result.ID,
			Slug:           result.Slug,
			APIKey:         result.ApiKey,
			Type:           EnvironmentType(result.Type),
			OrganizationID: result.OrganizationID,
			ProjectID:      result.ProjectID,
			OrgMemberID:    result.OrgMemberID,
			CreatedAt:      result.CreatedAt.Time,
			UpdatedAt:      result.UpdatedAt.Time,
		},
		ProjectID:   result.ProjectID_2,
		ProjectSlug: result.ProjectSlug,
		ProjectName: result.ProjectName,
		OrgID:       result.OrgID,
		OrgSlug:     result.OrgSlug,
		OrgTitle:    result.OrgTitle,
	}, nil
}

// AuthenticatePersonalAccessToken authenticates personal access token using apiauth queries
func (r *repository) AuthenticatePersonalAccessToken(ctx context.Context, token string) (*PersonalAccessTokenResult, error) {
	pat, err := r.apiAuthQueries.FindPersonalAccessToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("personal access token not found or expired: %w", err)
	}

	// Update last used timestamp
	if err := r.apiAuthQueries.UpdatePersonalTokenLastUsed(ctx, pat.ID); err != nil {
		// Log error but don't fail authentication
		// In production, you might want to use proper logging here
		fmt.Printf("Failed to update personal token last used: %v\n", err)
	}

	userID := ""
	if pat.UserID.Valid {
		userID = uuid.UUID(pat.UserID.Bytes).String()
	}

	return &PersonalAccessTokenResult{
		Token:  &pat,
		UserID: userID,
	}, nil
}

// AuthenticateOrganizationAccessToken authenticates organization access token using apiauth queries
func (r *repository) AuthenticateOrganizationAccessToken(ctx context.Context, token string) (*OrganizationAccessTokenResult, error) {
	oat, err := r.apiAuthQueries.FindOrganizationAccessToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("organization access token not found or expired: %w", err)
	}

	// Update last used timestamp
	if err := r.apiAuthQueries.UpdateOrgTokenLastUsed(ctx, oat.ID); err != nil {
		// Log error but don't fail authentication
		fmt.Printf("Failed to update org token last used: %v\n", err)
	}

	orgID := ""
	if oat.OrganizationID.Valid {
		orgID = uuid.UUID(oat.OrganizationID.Bytes).String()
	}

	return &OrganizationAccessTokenResult{
		Token: &oat,
		OrgID: orgID,
	}, nil
}
