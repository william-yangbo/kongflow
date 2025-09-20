-- 008_events_service.sql
-- Events Service 增量迁移，扩展现有 event_records 表并创建 event_dispatchers 表

-- 为现有 event_records 表添加缺失的字段
ALTER TABLE event_records 
ADD COLUMN IF NOT EXISTS external_account_id UUID,
ADD COLUMN IF NOT EXISTS deliver_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ;

-- 添加外键约束（如果 external_accounts 表存在）
-- ALTER TABLE event_records 
-- ADD CONSTRAINT fk_event_records_external_account 
-- FOREIGN KEY (external_account_id) REFERENCES external_accounts(id) ON DELETE SET NULL;

-- 创建事件调度器表，对齐 trigger.dev EventDispatcher
CREATE TABLE IF NOT EXISTS event_dispatchers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event TEXT NOT NULL,
    source TEXT NOT NULL,
    payload_filter JSONB,
    context_filter JSONB,
    manual BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- 可调度对象，对齐 trigger.dev dispatchable 设计
    dispatchable_id TEXT NOT NULL,
    dispatchable JSONB NOT NULL,
    
    -- 状态控制
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- 关联字段
    environment_id UUID NOT NULL,
    
    -- 元数据
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 唯一约束：确保同一环境下可调度对象唯一
    CONSTRAINT event_dispatchers_dispatchable_id_environment_id_key UNIQUE(dispatchable_id, environment_id),
    
    -- 外键约束
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE
);

-- 创建新的索引

-- EventRecord 新增字段索引
CREATE INDEX IF NOT EXISTS idx_event_records_deliver_at 
ON event_records(deliver_at) WHERE delivered_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_event_records_external_account 
ON event_records(external_account_id) WHERE external_account_id IS NOT NULL;

-- EventDispatcher 索引
CREATE INDEX IF NOT EXISTS idx_event_dispatchers_environment_id 
ON event_dispatchers(environment_id);

CREATE INDEX IF NOT EXISTS idx_event_dispatchers_event_source_enabled 
ON event_dispatchers(event, source, enabled) WHERE enabled = TRUE;

CREATE INDEX IF NOT EXISTS idx_event_dispatchers_dispatchable_id 
ON event_dispatchers(dispatchable_id);

-- EventDispatcher 更新触发器
CREATE TRIGGER update_event_dispatchers_updated_at 
BEFORE UPDATE ON event_dispatchers  
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 添加注释说明
COMMENT ON TABLE event_dispatchers IS 'Events 事件调度器表，对齐 trigger.dev EventDispatcher 模型';
COMMENT ON COLUMN event_records.deliver_at IS '计划投递时间，支持延迟投递';
COMMENT ON COLUMN event_records.delivered_at IS '实际投递时间，NULL表示未投递';
COMMENT ON COLUMN event_records.external_account_id IS '关联的外部账户ID';
COMMENT ON COLUMN event_dispatchers.dispatchable IS '可调度对象的JSON定义，包含类型和ID信息';
COMMENT ON COLUMN event_dispatchers.payload_filter IS '事件负载过滤规则的JSON定义';
COMMENT ON COLUMN event_dispatchers.context_filter IS '事件上下文过滤规则的JSON定义';