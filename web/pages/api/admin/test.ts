import type { NextApiRequest, NextApiResponse } from 'next'
import { withAdminAuth } from '../../lib/adminAuth'
import { getEntropyBridge } from '../../lib/rust-entropy-bridge'

type ResponseData = {
  message: string
  timestamp: number
  entropyBridgeStatus?: any
  secretInfo?: any
}

/**
 * Admin-only API endpoint demonstrating dynamic entropy-based authentication
 * GET /api/admin/test
 */
export default withAdminAuth(async (
  req: NextApiRequest,
  res: NextApiResponse<ResponseData>
) => {
  if (req.method !== 'GET') {
    return res.status(405).json({
      message: 'Method not allowed',
      timestamp: Date.now()
    })
  }

  try {
    // Get entropy bridge status
    const entropyBridge = getEntropyBridge()
    const bridgeStatus = entropyBridge.getStatus()

    // Get current admin secret info (for debugging - don't expose in production)
    const secretInfo = process.env.NODE_ENV === 'development' ? {
      hasCachedSecret: !!(global as any).cachedAdminSecret,
      lastGenerated: (global as any).secretLastGenerated || null,
      bridgeAvailable: bridgeStatus.rustAvailable
    } : undefined

    res.status(200).json({
      message: 'Admin authentication successful! 🎉',
      timestamp: Date.now(),
      entropyBridgeStatus: bridgeStatus,
      secretInfo
    })
  } catch (error) {
    console.error('Admin test endpoint error:', error)
    res.status(500).json({
      message: 'Internal server error',
      timestamp: Date.now()
    })
  }
})
