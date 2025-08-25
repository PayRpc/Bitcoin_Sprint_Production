#!/usr/bin/env node
// Simple script to reset requestsToday and blocksToday for all keys. Intended to be run daily by cron or scheduled task.
const { PrismaClient } = require('@prisma/client');
const prisma = new PrismaClient();

async function main() {
  console.log('[reset-daily-counters] Starting reset...');
  const result = await prisma.apiKey.updateMany({ data: { requestsToday: 0, blocksToday: 0 } });
  console.log('[reset-daily-counters] Reset complete:', result);
  await prisma.$disconnect();
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
