import type { NextApiRequest, NextApiResponse } from 'next';
import { generateTierApiKey } from "../../lib/generateKey";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  console.log('[SIMPLE SIGNUP] Request method:', req.method);
  console.log('[SIMPLE SIGNUP] Request body:', req.body);
  
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    const { email, tier } = req.body;
    
    // Basic validation
    if (!email || !email.includes('@')) {
      return res.status(400).json({ error: 'Valid email required' });
    }

    if (!tier) {
      return res.status(400).json({ error: 'Tier required' });
    }

    // Generate a test API key with tier-specific prefix
    const key = generateTierApiKey(tier);
    
    console.log('[SIMPLE SIGNUP] Generated key:', key);
    
    // Return without database for now
    return res.status(200).json({
      id: 'test-' + Date.now(),
      key: key,
      tier: tier,
      expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
      message: 'Test key generated (not persisted)'
    });

  } catch (error) {
    console.error('[SIMPLE SIGNUP] Error:', error);
    return res.status(500).json({ 
      error: 'Internal server error',
      details: error instanceof Error ? error.message : 'Unknown error'
    });
  }
}
