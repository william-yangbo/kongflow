-- 创建 ApiIntegrationVote 表
-- 严格对齐 trigger.dev 的数据模型
CREATE TABLE "public"."ApiIntegrationVote" (
    "id" TEXT NOT NULL,
    "apiIdentifier" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "ApiIntegrationVote_pkey" PRIMARY KEY ("id")
);

-- 创建唯一索引（防止重复投票）
-- 完全对齐 trigger.dev 的约束逻辑
CREATE UNIQUE INDEX "ApiIntegrationVote_apiIdentifier_userId_key"
ON "public"."ApiIntegrationVote"("apiIdentifier", "userId");

-- 添加注释说明
COMMENT ON TABLE "public"."ApiIntegrationVote" IS 'API集成投票表，与trigger.dev模型100%对齐';
COMMENT ON COLUMN "public"."ApiIntegrationVote"."apiIdentifier" IS 'API标识符，如"github", "slack"等';
COMMENT ON COLUMN "public"."ApiIntegrationVote"."userId" IS '投票用户ID，关联用户表';
COMMENT ON COLUMN "public"."ApiIntegrationVote"."createdAt" IS '投票创建时间';
COMMENT ON COLUMN "public"."ApiIntegrationVote"."updatedAt" IS '投票更新时间';

-- 注意：外键约束在生产环境中根据User表存在情况添加
-- ALTER TABLE "public"."ApiIntegrationVote"
-- ADD CONSTRAINT "ApiIntegrationVote_userId_fkey"
-- FOREIGN KEY ("userId") REFERENCES "public"."User"("id")
-- ON DELETE CASCADE ON UPDATE CASCADE;