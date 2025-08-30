import type { NextApiRequest, NextApiResponse } from 'next'
import { withApiKeyAuth } from '../../lib/apiKeyAuth'
import { renewApiKey } from '../../lib/generateKey'

async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' })
  const auth = req.headers.authorization || ''
  const parts = String(auth).split(' ')
  if (parts.length !== 2 || parts[0] !== 'Bearer') {
    return res.status(401).json({ error: 'Authentication required' })
  }
  const token = parts[1]
  if (!token) {
    return res.status(401).json({ error: 'Invalid token' })
  }
  const extendedDays = Number(req.body?.days) || 30
  const updated = await renewApiKey(token, extendedDays)
  if (!updated) return res.status(400).json({ error: 'Failed to renew key' })
  return res.status(200).json({ ok: true, expiresAt: updated.expiresAt })
}

export default withApiKeyAuth(async (req, res) => handler(req, res), { updateUsage: false })
