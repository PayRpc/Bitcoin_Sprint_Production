import crypto from "crypto"

export function generateApiKey(): string {
  // 256-bit secure random key
  return crypto.randomBytes(32).toString("base64url")
}
