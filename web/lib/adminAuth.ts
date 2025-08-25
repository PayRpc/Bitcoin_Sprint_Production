import crypto from "crypto"
import type { NextApiRequest, NextApiResponse } from "next"

/**
 * Verify admin authentication via x-admin-secret header.
 * Uses constant-time comparison to prevent timing attacks.
 */
export function requireAdminAuth(req: NextApiRequest): boolean {
  const adminSecret = process.env.ADMIN_SECRET
  if (!adminSecret) {
    // Fail closed if secret is not configured
    return false
  }

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
    if (!requireAdminAuth(req)) {
      return res.status(401).json({
        error: "Unauthorized",
        message: "Valid x-admin-secret header required"
      } as T)
    }

    return handler(req, res)
  }
}
