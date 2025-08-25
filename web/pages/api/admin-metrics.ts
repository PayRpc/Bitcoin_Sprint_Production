import { PrismaClient } from '@prisma/client'
import type { NextApiRequest, NextApiResponse } from 'next'
import { requireAdminAuth } from '../../lib/adminAuth'

const prisma = new PrismaClient()

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (!requireAdminAuth(req)) return res.status(403).json({ error: 'Forbidden' })
  const active = await prisma.apiKey.count({ where: { revoked: false, expiresAt: { gt: new Date() } } })
  const total = await prisma.apiKey.count()
  return res.status(200).json({ active, total })
}
