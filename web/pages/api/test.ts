import type { NextApiRequest, NextApiResponse } from 'next';

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  // Simple test endpoint to check if API routes are working
  if (req.method === 'GET') {
    return res.status(200).json({ 
      message: 'Test endpoint working',
      timestamp: new Date().toISOString(),
      method: req.method,
      url: req.url
    });
  }
  
  if (req.method === 'POST') {
    return res.status(200).json({
      message: 'POST endpoint working',
      body: req.body,
      timestamp: new Date().toISOString()
    });
  }
  
  return res.status(405).json({ error: 'Method not allowed' });
}
