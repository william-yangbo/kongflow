-- 007_event_records.sql
-- EventRecord 表结构，支持 TestJob 功能的事件记录创建和管理

-- 事件记录表，对齐 trigger.dev 的 EventRecord 模型
CREATE TABLE event_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(500) NOT NULL,
    name VARCHAR(255) NOT NULL,
    source VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    context JSONB NOT NULL DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    environment_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    project_id UUID NOT NULL,
    is_test BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(event_id, environment_id),
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- 事件记录索引
CREATE INDEX idx_event_records_event_id ON event_records(event_id);
CREATE INDEX idx_event_records_environment ON event_records(environment_id);
CREATE INDEX idx_event_records_name_source ON event_records(name, source);
CREATE INDEX idx_event_records_timestamp ON event_records(timestamp DESC);
CREATE INDEX idx_event_records_is_test ON event_records(is_test);
CREATE INDEX idx_event_records_org_project ON event_records(organization_id, project_id);

-- 为 JSONB 字段添加 GIN 索引
CREATE INDEX idx_event_records_payload ON event_records USING GIN (payload);
CREATE INDEX idx_event_records_context ON event_records USING GIN (context);

-- 更新时间触发器
CREATE TRIGGER update_event_records_updated_at BEFORE UPDATE ON event_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释说明
COMMENT ON TABLE event_records IS '事件记录表，存储测试事件和实际事件记录';
COMMENT ON COLUMN event_records.event_id IS '事件唯一标识符';
COMMENT ON COLUMN event_records.name IS '事件名称';
COMMENT ON COLUMN event_records.source IS '事件源，如 trigger.dev';
COMMENT ON COLUMN event_records.payload IS '事件负载数据 JSON';
COMMENT ON COLUMN event_records.context IS '事件上下文数据 JSON';
COMMENT ON COLUMN event_records.is_test IS '是否为测试事件';