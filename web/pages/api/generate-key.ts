import { goApiClient } from '../../lib/goApiClient';
import type { NextApiRequest, NextApiResponse } from 'next';

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    const response = await goApiClient.generateKey();
    
    if (response.error) {
      return res.status(response.status).json({ error: response.error });
    }

    res.status(200).json(response.data);
  } catch (error: any) {
    console.error('Generate key failed:', error);
    res.status(500).json({ 
      error: error.message || 'Failed to generate key'
    });
  }
}
