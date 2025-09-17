-- CreateTable SecretStore (trigger.dev compatible)
CREATE TABLE "SecretStore" (
    "key" TEXT NOT NULL,
    "value" JSONB NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "SecretStore_key_key" ON "SecretStore"("key");