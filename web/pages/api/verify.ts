import type { NextApiRequest, NextApiResponse } from 'next'
import { verifyKey } from '../../lib/verifyKey'

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const auth = req.headers.authorization || ''
  const parts = String(auth).split(' ')
  if (parts.length !== 2 || parts[0] !== 'Bearer') {
    return res.status(401).json({ ok: false })
  }
  const key = parts[1]
  if (!key) {
    return res.status(401).json({ ok: false })
  }
  const v = await verifyKey(key)
  if (!v.ok) return res.status(401).json({ ok: false, reason: v.revoked ? 'revoked' : v.expired ? 'expired' : 'unknown' })
  return res.status(200).json({ ok: true, tier: v.tier })
}
