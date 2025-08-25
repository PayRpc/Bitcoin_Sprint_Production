-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_ApiKey" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "key" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "company" TEXT,
    "tier" TEXT NOT NULL DEFAULT 'FREE',
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expiresAt" DATETIME NOT NULL,
    "revoked" BOOLEAN NOT NULL DEFAULT false,
    "lastUsedAt" DATETIME,
    "requests" INTEGER NOT NULL DEFAULT 0,
    "requestsToday" INTEGER NOT NULL DEFAULT 0,
    "blocksToday" INTEGER NOT NULL DEFAULT 0
);
INSERT INTO "new_ApiKey" ("blocksToday", "company", "createdAt", "email", "expiresAt", "id", "key", "lastUsedAt", "requests", "revoked", "tier") SELECT "blocksToday", "company", "createdAt", "email", "expiresAt", "id", "key", "lastUsedAt", "requests", "revoked", "tier" FROM "ApiKey";
DROP TABLE "ApiKey";
ALTER TABLE "new_ApiKey" RENAME TO "ApiKey";
CREATE UNIQUE INDEX "ApiKey_key_key" ON "ApiKey"("key");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
