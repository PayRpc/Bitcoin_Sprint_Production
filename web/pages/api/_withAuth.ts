import { authMiddleware, type AuthenticatedRequest } from '@/lib/auth';
import type { NextApiHandler, NextApiRequest, NextApiResponse } from 'next';

type AuthenticatedHandler = (req: AuthenticatedRequest, res: NextApiResponse) => void | Promise<void>;

export function withAuth(handler: AuthenticatedHandler): NextApiHandler {
  return async (req: NextApiRequest, res: NextApiResponse) => {
    let finished = false;
    
    await authMiddleware(req, res, () => { 
      finished = true; 
    });
    
    if (!finished) return;
    
    return handler(req as AuthenticatedRequest, res);
  };
}
