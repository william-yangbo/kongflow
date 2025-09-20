-- 009_external_accounts.sql
-- External Accounts 表，对齐 trigger.dev ExternalAccount 模型

-- 创建外部账户表
CREATE TABLE external_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier VARCHAR(255) NOT NULL,
    metadata JSONB,
    organization_id UUID NOT NULL,
    environment_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- 约束条件，对齐 trigger.dev
    UNIQUE(environment_id, identifier),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE
);

-- 索引优化
CREATE INDEX idx_external_accounts_org_id ON external_accounts(organization_id);
CREATE INDEX idx_external_accounts_env_identifier ON external_accounts(environment_id, identifier);

-- 启用 event_records 表的外键约束
ALTER TABLE event_records 
ADD CONSTRAINT fk_event_records_external_account 
FOREIGN KEY (external_account_id) REFERENCES external_accounts(id) ON DELETE SET NULL;

-- 更新时间触发器
CREATE TRIGGER update_external_accounts_updated_at 
BEFORE UPDATE ON external_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释说明
COMMENT ON TABLE external_accounts IS '外部账户表，对齐 trigger.dev ExternalAccount 模型';
COMMENT ON COLUMN external_accounts.identifier IS '外部账户标识符，在环境内唯一';
COMMENT ON COLUMN external_accounts.metadata IS '外部账户元数据 JSON';