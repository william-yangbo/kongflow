-- 005_endpoints_service.sql
-- Endpoints Service Migration
-- 对齐 trigger.dev Endpoint 和 EndpointIndex 模型

-- 创建 endpoints 表 (对齐 trigger.dev Endpoint 模型)
CREATE TABLE endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) NOT NULL,
    title TEXT,
    url TEXT NOT NULL,
    indexing_hook_identifier VARCHAR(10) NOT NULL,
    indexing_stats JSONB NOT NULL DEFAULT '{}',

    -- 关联关系 (严格对齐 trigger.dev)
    environment_id UUID NOT NULL REFERENCES runtime_environments(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- 约束 (对齐 trigger.dev @@unique([environmentId, slug]))
    UNIQUE(environment_id, slug)
);

-- 创建 endpoint_indexes 表 (对齐 trigger.dev EndpointIndex 模型)
CREATE TABLE endpoint_indexes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,

    -- 统计和数据 (JSONB 对齐 trigger.dev)
    stats JSONB NOT NULL DEFAULT '{}',
    data JSONB NOT NULL DEFAULT '{}',

    -- 索引来源 (对齐 EndpointIndexSource 枚举)
    source VARCHAR(50) NOT NULL DEFAULT 'MANUAL'
        CHECK (source IN ('MANUAL', 'API', 'INTERNAL', 'HOOK')),
    source_data JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    reason TEXT,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 性能优化索引
CREATE INDEX idx_endpoints_environment_id ON endpoints(environment_id);
CREATE INDEX idx_endpoints_organization_id ON endpoints(organization_id);
CREATE INDEX idx_endpoints_project_id ON endpoints(project_id);
CREATE INDEX idx_endpoints_slug ON endpoints(environment_id, slug);

CREATE INDEX idx_endpoint_indexes_endpoint_id ON endpoint_indexes(endpoint_id);
CREATE INDEX idx_endpoint_indexes_source ON endpoint_indexes(source);
CREATE INDEX idx_endpoint_indexes_created_at ON endpoint_indexes(created_at);

-- 添加更新时间戳触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_endpoints_updated_at BEFORE UPDATE ON endpoints 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_endpoint_indexes_updated_at BEFORE UPDATE ON endpoint_indexes 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();