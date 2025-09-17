-- CreateTable SecretStore (trigger.dev compatible)
CREATE TABLE "SecretStore" (
    "key" TEXT NOT NULL,
    "value" BYTEA NOT NULL,  -- 修复：统一使用BYTEA存储加密数据，比JSONB更安全
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "SecretStore_key_key" ON "SecretStore"("key");