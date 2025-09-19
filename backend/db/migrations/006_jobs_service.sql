-- 006_jobs_service.sql
-- Jobs Service 数据库表结构，严格对齐 trigger.dev 的数据模型

-- Job 作业主体表
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    internal BOOLEAN NOT NULL DEFAULT false,
    organization_id UUID NOT NULL,
    project_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(project_id, slug),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- 索引优化
CREATE INDEX idx_jobs_project_slug ON jobs(project_id, slug);
CREATE INDEX idx_jobs_organization ON jobs(organization_id);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);

-- Job 开始位置枚举类型
CREATE TYPE job_start_position AS ENUM ('INITIAL', 'LATEST');

-- Job 队列表
CREATE TABLE job_queues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    environment_id UUID NOT NULL,
    job_count INTEGER NOT NULL DEFAULT 0,
    max_jobs INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(environment_id, name),
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE
);

-- 队列索引
CREATE INDEX idx_job_queues_environment ON job_queues(environment_id);
CREATE INDEX idx_job_queues_name ON job_queues(environment_id, name);

-- Job 版本表
CREATE TABLE job_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    version VARCHAR(100) NOT NULL,
    event_specification JSONB NOT NULL,
    properties JSONB,
    endpoint_id UUID NOT NULL,
    environment_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    project_id UUID NOT NULL,
    queue_id UUID NOT NULL,
    start_position job_start_position NOT NULL DEFAULT 'INITIAL',
    preprocess_runs BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(job_id, version, environment_id),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE,
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (queue_id) REFERENCES job_queues(id) ON DELETE RESTRICT
);

-- 版本索引
CREATE INDEX idx_job_versions_job ON job_versions(job_id);
CREATE INDEX idx_job_versions_environment ON job_versions(environment_id);
CREATE INDEX idx_job_versions_endpoint ON job_versions(endpoint_id);
CREATE INDEX idx_job_versions_version ON job_versions(job_id, version);

-- Job 别名表
CREATE TABLE job_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    version_id UUID NOT NULL,
    environment_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT 'latest',
    value VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(job_id, environment_id, name),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES job_versions(id) ON DELETE CASCADE,
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE
);

-- 别名索引
CREATE INDEX idx_job_aliases_job_env ON job_aliases(job_id, environment_id);
CREATE INDEX idx_job_aliases_name ON job_aliases(environment_id, name);

-- 事件示例表
CREATE TABLE event_examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_version_id UUID NOT NULL,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(500) NOT NULL,
    icon VARCHAR(255),
    payload JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(slug, job_version_id),
    FOREIGN KEY (job_version_id) REFERENCES job_versions(id) ON DELETE CASCADE
);

-- 事件示例索引
CREATE INDEX idx_event_examples_job_version ON event_examples(job_version_id);
CREATE INDEX idx_event_examples_slug ON event_examples(slug);

-- 更新时间触发器函数（如果不存在）
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为所有表添加更新时间触发器
CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_job_queues_updated_at BEFORE UPDATE ON job_queues
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_job_versions_updated_at BEFORE UPDATE ON job_versions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_job_aliases_updated_at BEFORE UPDATE ON job_aliases
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_event_examples_updated_at BEFORE UPDATE ON event_examples
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 为 JSONB 字段添加 GIN 索引以优化查询性能
CREATE INDEX idx_job_versions_event_specification ON job_versions USING GIN (event_specification);
CREATE INDEX idx_job_versions_properties ON job_versions USING GIN (properties);
CREATE INDEX idx_event_examples_payload ON event_examples USING GIN (payload);

-- 注释说明
COMMENT ON TABLE jobs IS 'Job 作业主体表，对齐 trigger.dev 的 Job 模型';
COMMENT ON TABLE job_queues IS 'Job 队列表，管理作业执行队列';
COMMENT ON TABLE job_versions IS 'Job 版本表，管理作业的不同版本';
COMMENT ON TABLE job_aliases IS 'Job 别名表，提供版本别名功能';
COMMENT ON TABLE event_examples IS '事件示例表，存储作业的事件示例数据';

COMMENT ON COLUMN jobs.slug IS '作业唯一标识符，在项目内唯一';
COMMENT ON COLUMN jobs.internal IS '是否为内部作业';
COMMENT ON COLUMN job_versions.event_specification IS '事件规范的 JSON 定义';
COMMENT ON COLUMN job_versions.properties IS '作业属性的 JSON 定义';
COMMENT ON COLUMN job_versions.start_position IS '作业开始位置：INITIAL 或 LATEST';
COMMENT ON COLUMN job_versions.preprocess_runs IS '是否需要预处理运行';