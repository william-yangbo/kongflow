-- Migration: 004_apiauth_service.sql
-- Description: Create ApiAuth service tables for authentication tokens
-- Author: Kong Flow Migration Team
-- Created: 2025-09-18

-- Personal Access Tokens table - trigger.dev PersonalAccessToken alignment
CREATE TABLE personal_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organization Access Tokens table - trigger.dev OrganizationAccessToken alignment
CREATE TABLE organization_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_personal_access_tokens_token ON personal_access_tokens(token);
CREATE INDEX idx_personal_access_tokens_user_id ON personal_access_tokens(user_id);
CREATE INDEX idx_personal_access_tokens_expires_at ON personal_access_tokens(expires_at);
CREATE INDEX idx_organization_access_tokens_token ON organization_access_tokens(token);
CREATE INDEX idx_organization_access_tokens_org_id ON organization_access_tokens(organization_id);
CREATE INDEX idx_organization_access_tokens_expires_at ON organization_access_tokens(expires_at);