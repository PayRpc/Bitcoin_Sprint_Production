import { authMiddleware, type AuthenticatedRequest } from '@/lib/auth';
import type { NextApiHandler, NextApiRequest, NextApiResponse } from 'next';

export function withAuth(handler: NextApiHandler): NextApiHandler {
  return async (req: NextApiRequest, res: NextApiResponse) => {
    let finished = false;
    
    await authMiddleware(req, res, () => { 
      finished = true; 
    });
    
    if (!finished) return;
    
    return handler(req as AuthenticatedRequest, res);
  };
}
