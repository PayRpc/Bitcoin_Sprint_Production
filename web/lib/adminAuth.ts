import type { NextApiRequest } from "next"

export function requireAdminAuth(req: NextApiRequest): boolean {
  const adminSecret = process.env.ADMIN_SECRET || ""
  const provided = req.headers["x-admin-secret"]
  if (Array.isArray(provided)) return provided[0] === adminSecret
  return provided === adminSecret
}
