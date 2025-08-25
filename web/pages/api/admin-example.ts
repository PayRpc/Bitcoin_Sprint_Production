import type { NextApiRequest, NextApiResponse } from 'next'
import { withAdminAuth } from '../../lib/adminAuth'

/**
 * Example admin-only API route using the withAdminAuth wrapper.
 * No need to manually check authentication - it's handled automatically.
 */
export default withAdminAuth(async function handler(req: NextApiRequest, res: NextApiResponse) {
  // This code only runs if x-admin-secret header is valid
  
  if (req.method === 'GET') {
    return res.status(200).json({
      message: "Admin access granted!",
      timestamp: new Date().toISOString(),
      adminInfo: {
        hasAccess: true,
        permissions: ["read", "write", "delete"]
      }
    })
  }

  res.setHeader('Allow', 'GET')
  return res.status(405).json({ error: 'Method not allowed' })
})
