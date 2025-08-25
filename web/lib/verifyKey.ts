import { PrismaClient } from '@prisma/client'

const prisma = new PrismaClient()

export async function verifyKey(key: string) {
  if (!key) return { ok: false }
  const rec = await prisma.apiKey.findUnique({ where: { key } })
  if (!rec) return { ok: false }
  const now = new Date()
  if (rec.revoked) return { ok: false, revoked: true }
  if (rec.expiresAt <= now) return { ok: false, expired: true }
  return { ok: true, tier: rec.tier, expiresAt: rec.expiresAt, email: rec.email, requests: rec.requests, requestsToday: (rec as any).requestsToday ?? 0 }
}
