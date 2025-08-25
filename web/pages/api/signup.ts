import { PrismaClient } from "@prisma/client"
import type { NextApiRequest, NextApiResponse } from "next"
import { generateTierApiKey } from "../../lib/generateKey"

// Prisma client singleton for Next.js dev hot-reload safety
declare global {
  // eslint-disable-next-line no-var
  var prisma: PrismaClient | undefined
}

const prisma: PrismaClient = global.prisma || new PrismaClient()
if (process.env.NODE_ENV !== "production") global.prisma = prisma

interface SignupRequest {
  email: string;
  company?: string;
  tier: string;
}

interface SignupResponse {
  id: string;
  key: string;
  tier: string;
  expiresAt: string;
}

interface ErrorResponse {
  error: string;
  message?: string;
}

/**
 * Public API endpoint for user signup and API key generation
 * No admin authentication required - this is the public signup flow
 */
export default async function handler(
  req: NextApiRequest, 
  res: NextApiResponse<SignupResponse | ErrorResponse>
) {
  console.log('[SIGNUP] Request received:', req.method, req.url);
  console.log('[SIGNUP] Request body:', req.body);
  
  // Ensure we always return JSON, even on early errors
  res.setHeader('Content-Type', 'application/json');
  
  if (req.method !== "POST") {
    console.log('[SIGNUP] Method not allowed:', req.method);
    return res.status(405).json({ error: "Method not allowed" })
  }

  // Validate content type
  if (req.headers['content-type'] !== 'application/json') {
    return res.status(400).json({
      error: "Invalid content type",
      message: "Expected application/json"
    });
  }

  try {
    const { email, company, tier } = (req.body || {}) as SignupRequest
    console.log('[SIGNUP] Parsed data:', { email, company, tier });
    
    // Validate email
    if (typeof email !== "string" || !email.includes("@")) {
      console.log('[SIGNUP] Invalid email:', email);
      return res.status(400).json({ 
        error: "Invalid email", 
        message: "Please provide a valid email address" 
      })
    }

    // Validate tier
    if (!tier || !["FREE", "PRO", "ENTERPRISE", "ENTERPRISE_PLUS"].includes(tier)) {
      console.log('[SIGNUP] Invalid tier:', tier);
      return res.status(400).json({ 
        error: "Invalid tier", 
        message: "Tier must be one of: FREE, PRO, ENTERPRISE, ENTERPRISE_PLUS" 
      })
    }

    // Generate API key and set expiration
    console.log('[SIGNUP] Generating API key for tier:', tier);
    const key = generateTierApiKey(tier)
    console.log('[SIGNUP] Generated key:', key.substring(0, 20) + '...');
    console.log('[SIGNUP] Generated key length:', key.length);
    const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000) // 30 days

    console.log('[SIGNUP] Creating database record...');
    // Create database record
    const record = await prisma.apiKey.create({
      data: { 
        key, 
        email, 
        company: company || null, 
        tier, 
        expiresAt 
      }
    })

    console.log('[SIGNUP] Database record created:', record.id);

    // Audit log
    console.log(`[AUDIT] API key created - ID: ${record.id}, Email: ${email}, Tier: ${tier}, Company: ${company || 'N/A'}`)

    // Return key details
    return res.status(201).json({
      id: record.id,
      key: record.key,
      tier: record.tier,
      expiresAt: record.expiresAt.toISOString()
    })

  } catch (error: any) {
    console.error("[ERROR] API key creation failed:", error)
    
    // Handle database constraint errors (e.g., duplicate email)
    if (error?.code === 'P2002') {
      return res.status(409).json({ 
        error: "Conflict", 
        message: "An API key for this email already exists" 
      })
    }

    // Handle any other errors gracefully
    return res.status(500).json({ 
      error: "Internal server error", 
      message: error?.message || "Failed to generate API key. Please try again." 
    })
  }
}
