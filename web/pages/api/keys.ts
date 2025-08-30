import { PrismaClient } from "@prisma/client"
import type { NextApiRequest, NextApiResponse } from "next"
import { withAdminAuth } from "../../lib/adminAuth"
import { generateApiKey } from "../../lib/generateKey"

// Prisma client singleton for Next.js dev hot-reload safety
declare global {
  // eslint-disable-next-line no-var
  var prisma: PrismaClient | undefined
}

const prisma: PrismaClient = global.prisma || new PrismaClient()
if (process.env.NODE_ENV !== "production") global.prisma = prisma

export default withAdminAuth(async function handler(req: NextApiRequest, res: NextApiResponse) {
  try {
    if (req.method === "POST") {
      const { email, company, tier } = (req.body || {}) as { email?: string; company?: string; tier?: string }
      if (typeof email !== "string" || !email.includes("@")) {
        return res.status(400).json({ error: "Valid email required" })
      }
      if (!tier || !["free", "pro", "enterprise"].includes(tier)) {
        return res.status(400).json({ error: "Invalid tier" })
      }

      const key = generateApiKey()
      const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000) // 30 days

      const record = await prisma.apiKey.create({
        data: { key, email, company: company || null, tier, expiresAt }
      })

      // Audit log - write to stdout (can be captured by aggregator)
      console.log(JSON.stringify({ event: 'api_key_issued', id: record.id, email: record.email, tier: record.tier, ts: new Date().toISOString() }))

      return res.status(200).json({
        id: record.id,
        key: record.key,
        tier: record.tier,
        expiresAt: record.expiresAt
      })
    }

    if (req.method === "GET") {
      const keys = await prisma.apiKey.findMany({ orderBy: { createdAt: "desc" } })
      // Avoid leaking raw key material in listings
      const safe = keys.map((k: any) => ({
        id: k.id,
        email: k.email,
        company: k.company,
        tier: k.tier,
        createdAt: k.createdAt,
        expiresAt: k.expiresAt,
        revoked: k.revoked
      }))
      return res.status(200).json(safe)
    }

    if (req.method === "DELETE") {
      const idParam = (req.query?.id as string | string[] | undefined)
      const id = Array.isArray(idParam) ? idParam[0] : idParam
      if (!id || typeof id !== "string") {
        return res.status(400).json({ error: "Key ID required" })
      }

  const rec = await prisma.apiKey.update({ where: { id }, data: { revoked: true } })
  console.log(JSON.stringify({ event: 'api_key_revoked', id: rec.id, email: rec.email, ts: new Date().toISOString() }))
  return res.status(200).json({ ok: true })
    }

    res.setHeader("Allow", "GET, POST, DELETE")
    return res.status(405).json({ error: "Method not allowed" })
  } catch (err) {
    // Do not leak internals
    return res.status(500).json({ error: "Internal server error" })
  }
})
