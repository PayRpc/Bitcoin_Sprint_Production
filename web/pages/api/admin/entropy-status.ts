import type { NextApiRequest, NextApiResponse } from 'next'
import { getEntropyBridge } from '../../../lib/rust-entropy-bridge'

type EntropyBridgeStatus = {
  available: boolean
  rustAvailable: boolean
  fallbackMode: boolean
  timestamp: number
}

type ResponseData = {
  status: EntropyBridgeStatus
  uptime: number
  lastSecretGenerated?: number
}

/**
 * Entropy Bridge Status API Endpoint
 * GET /api/admin/entropy-status
 *
 * Returns the current status of the entropy bridge system
 */
export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<ResponseData>
) {
  if (req.method !== 'GET') {
    return res.status(405).json({
      status: {
        available: false,
        rustAvailable: false,
        fallbackMode: true,
        timestamp: Date.now()
      },
      uptime: process.uptime()
    })
  }

  try {
    const entropyBridge = getEntropyBridge()
    const bridgeStatus = entropyBridge.getStatus()

    const response: ResponseData = {
      status: {
        ...bridgeStatus,
        timestamp: Date.now()
      },
      uptime: process.uptime(),
      lastSecretGenerated: (global as any).secretLastGenerated || undefined
    }

    res.status(200).json(response)
  } catch (error) {
    console.error('Error getting entropy bridge status:', error)
    res.status(500).json({
      status: {
        available: false,
        rustAvailable: false,
        fallbackMode: true,
        timestamp: Date.now()
      },
      uptime: process.uptime()
    })
  }
}
