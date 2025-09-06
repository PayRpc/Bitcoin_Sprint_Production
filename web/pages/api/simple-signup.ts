import type { NextApiRequest, NextApiResponse } from 'next';
import { generateTierApiKey } from "../../lib/generateKey";
import { PrismaClient } from "@prisma/client";

// Prisma client singleton for Next.js dev hot-reload safety
declare global {
  // eslint-disable-next-line no-var
  var prisma: PrismaClient | undefined
}

const prisma: PrismaClient = global.prisma || new PrismaClient()
if (process.env.NODE_ENV !== "production") global.prisma = prisma

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  console.log('[SIMPLE SIGNUP] Request method:', req.method);
  console.log('[SIMPLE SIGNUP] Request body:', req.body);
  
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    const { email, company, tier } = req.body;
    
    // Basic validation
    if (!email || !email.includes('@')) {
      return res.status(400).json({ error: 'Valid email required' });
    }

    if (!tier) {
      return res.status(400).json({ error: 'Tier required' });
    }

    // Generate a secure API key with tier-specific prefix
    const key = generateTierApiKey(tier);
    
    console.log('[SIMPLE SIGNUP] Generated key for tier:', tier);
    
    // Calculate expiration date (1 year from now)
    const expiresAt = new Date();
    expiresAt.setFullYear(expiresAt.getFullYear() + 1);
    
    // Save to database
    const apiKeyRecord = await prisma.apiKey.create({
      data: {
        key: key,
        email: email,
        company: company || null,
        tier: tier,
        expiresAt: expiresAt,
        revoked: false,
        requests: 0,
        requestsToday: 0,
        blocksToday: 0
      }
    });
    
    console.log('[SIMPLE SIGNUP] Persisted API key with ID:', apiKeyRecord.id);
    
    // Also write to Go backend shared data for immediate availability
    await writeKeyToGoBackend(apiKeyRecord);
    
    // Return success response
    return res.status(200).json({
      id: apiKeyRecord.id,
      key: apiKeyRecord.key,
      tier: apiKeyRecord.tier,
      expiresAt: apiKeyRecord.expiresAt.toISOString(),
      message: 'API key generated and activated'
    });

  } catch (error) {
    console.error('[SIMPLE SIGNUP] Error:', error);
    return res.status(500).json({ 
      error: 'Internal server error',
      details: error instanceof Error ? error.message : 'Unknown error'
    });
  }
}

// Write API key to shared location for Go backend
async function writeKeyToGoBackend(apiKey: any) {
  try {
    const fs = require('fs').promises;
    const path = require('path');
    
    // Write to shared data directory that Go backend can read
    const sharedDataPath = path.join(process.cwd(), '../data/api_keys.json');
    
    let existingKeys = [];
    try {
      const existingData = await fs.readFile(sharedDataPath, 'utf8');
      existingKeys = JSON.parse(existingData);
    } catch (err) {
      // File doesn't exist yet, start with empty array
      existingKeys = [];
    }
    
    // Add new key
    existingKeys.push({
      id: apiKey.id,
      key: apiKey.key,
      email: apiKey.email,
      company: apiKey.company,
      tier: apiKey.tier,
      created_at: apiKey.createdAt.toISOString(),
      expires_at: apiKey.expiresAt.toISOString(),
      revoked: apiKey.revoked,
      last_used_at: null,
      requests: apiKey.requests,
      requests_today: apiKey.requestsToday,
      blocks_today: apiKey.blocksToday
    });
    
    // Ensure directory exists
    await fs.mkdir(path.dirname(sharedDataPath), { recursive: true });
    
    // Write updated keys
    await fs.writeFile(sharedDataPath, JSON.stringify(existingKeys, null, 2));
    
    console.log('[SIMPLE SIGNUP] Synced key to Go backend data store');
    
  } catch (error) {
    console.error('[SIMPLE SIGNUP] Failed to sync to Go backend:', error);
    // Don't fail the API call if sync fails
  }
}
