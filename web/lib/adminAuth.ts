import crypto from "crypto"
import type { NextApiRequest, NextApiResponse } from "next"
import { generateAdminSecret, isEntropyBridgeAvailable } from "./rust-entropy-bridge"
import { recordEntropySecretGeneration } from "./prometheus"

// Cache for the admin secret to avoid regenerating on every request
let cachedAdminSecret: string | null = null
let secretLastGenerated: number = 0
const SECRET_CACHE_DURATION = 1000 * 60 * 60 // 1 hour

/**
 * Get the current admin secret, either from cache or generate new one
 */
async function getAdminSecret(): Promise<string> {
  const now = Date.now()

  // Return cached secret if it's still valid
  if (cachedAdminSecret && (now - secretLastGenerated) < SECRET_CACHE_DURATION) {
    return cachedAdminSecret
  }

  try {
    // Generate new secret using enterprise entropy
    const startTime = Date.now()
    const newSecret = await generateAdminSecret('base64')
    const duration = (Date.now() - startTime) / 1000 // Convert to seconds

    cachedAdminSecret = newSecret
    secretLastGenerated = Date.now()

    // Record metrics
    const source = isEntropyBridgeAvailable() ? 'rust' : 'nodejs'
    recordEntropySecretGeneration(source, 'base64', duration)

    console.log(`🔐 Admin secret ${source === 'rust' ? 'generated with Rust entropy' : 'generated with fallback entropy'} (${duration.toFixed(4)}s)`)
    return newSecret
  } catch (error) {
    console.error('Failed to generate admin secret:', error)

    // Fallback to environment variable if generation fails
    const envSecret = process.env.ADMIN_SECRET
    if (envSecret) {
      console.warn('⚠️ Using fallback ADMIN_SECRET from environment')
      cachedAdminSecret = envSecret
      secretLastGenerated = Date.now()

      // Record environment fallback metrics
      recordEntropySecretGeneration('env', 'base64', 0)
      return envSecret
    }

    // Final fallback: generate a temporary secret
    console.error('❌ No admin secret available, authentication will fail')
    throw new Error('Admin secret generation failed and no fallback available')
  }
}

/**
 * Verify admin authentication via x-admin-secret header.
 * Uses constant-time comparison to prevent timing attacks.
 * Admin secret is dynamically generated using enterprise entropy.
 */
export async function requireAdminAuth(req: NextApiRequest): Promise<boolean> {
  try {
    const adminSecret = await getAdminSecret()

    const provided = Array.isArray(req.headers["x-admin-secret"])
      ? req.headers["x-admin-secret"][0]
      : req.headers["x-admin-secret"]

    if (typeof provided !== "string") {
      return false
    }

    // Convert to Buffers for timing-safe compare
    const secretBuf = Buffer.from(adminSecret, "utf8")
    const providedBuf = Buffer.from(provided, "utf8")

    // Quick fail if lengths differ (still safe)
    if (secretBuf.length !== providedBuf.length) {
      return false
    }

    return crypto.timingSafeEqual(secretBuf, providedBuf)
  } catch (error) {
    console.error('Admin auth error:', error)
    return false
  }
}

/**
 * Higher-order function that wraps Next.js API handlers with admin authentication.
 * Returns 401 Unauthorized if authentication fails, otherwise calls the handler.
 *
 * @example
 * export default withAdminAuth(async (req, res) => {
 *   // This handler only runs if admin auth succeeds
 *   res.json({ message: "Admin-only data" })
 * })
 */
export function withAdminAuth<T = any>(
  handler: (req: NextApiRequest, res: NextApiResponse<T>) => Promise<void> | void
) {
  return async (req: NextApiRequest, res: NextApiResponse<T>) => {
    try {
      const isAuthenticated = await requireAdminAuth(req)

      if (!isAuthenticated) {
        return res.status(401).json({
          error: "Unauthorized",
          message: "Valid x-admin-secret header required"
        } as T)
      }

      return handler(req, res)
    } catch (error) {
      console.error('Admin auth middleware error:', error)
      return res.status(500).json({
        error: "Internal Server Error",
        message: "Authentication system error"
      } as T)
    }
  }
}

/**
 * Synchronous version for backward compatibility (less secure)
 * @deprecated Use withAdminAuth for new implementations
 */
export function withAdminAuthSync<T = any>(
  handler: (req: NextApiRequest, res: NextApiResponse<T>) => Promise<void> | void
) {
  return async (req: NextApiRequest, res: NextApiResponse<T>) => {
    // For backward compatibility, try to use cached secret if available
    if (cachedAdminSecret) {
      const provided = Array.isArray(req.headers["x-admin-secret"])
        ? req.headers["x-admin-secret"][0]
        : req.headers["x-admin-secret"]

      if (typeof provided === "string") {
        const secretBuf = Buffer.from(cachedAdminSecret, "utf8")
        const providedBuf = Buffer.from(provided, "utf8")

        if (secretBuf.length === providedBuf.length && crypto.timingSafeEqual(secretBuf, providedBuf)) {
          return handler(req, res)
        }
      }
    }

    return res.status(401).json({
      error: "Unauthorized",
      message: "Valid x-admin-secret header required"
    } as T)
  }
}
