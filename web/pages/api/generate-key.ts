import type { NextApiRequest, NextApiResponse } from 'next';

type Data = {
  error: string;
  message: string;
  redirectTo?: string;
}

/**
 * @deprecated This endpoint has been replaced by /api/keys
 * @description Legacy API key generation endpoint - redirects to new implementation
 */
export default function handler(req: NextApiRequest, res: NextApiResponse<Data>) {
  return res.status(410).json({
    error: 'Endpoint Deprecated',
    message: 'This endpoint has been replaced. Please use /api/keys instead for API key generation with proper persistence and management.',
    redirectTo: '/api/keys'
  });
}
