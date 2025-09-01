import { authMiddleware, type AuthenticatedRequest } from '@/lib/auth';
import type { NextApiHandler, NextApiRequest, NextApiResponse } from 'next';

type AuthenticatedHandler = (req: AuthenticatedRequest, res: NextApiResponse) => void | Promise<void>;

export function withAuth(handler: AuthenticatedHandler): NextApiHandler {
  return async (req: NextApiRequest, res: NextApiResponse) => {
    // Allow unauthenticated access in development mode
    if (process.env.NODE_ENV === 'development') {
      (req as any).apiKey = {
        id: 'dev-key',
        key: 'bitcoin-sprint-dev-key-2025',
        tier: 'ENTERPRISE',
        email: 'dev@bitcoin-sprint.com',
        company: 'Bitcoin Sprint Dev',
        requests: Math.floor(Math.random() * 1000),
        blocksToday: Math.floor(Math.random() * 100)
      };
      return handler(req as AuthenticatedRequest, res);
    }

    let finished = false;

    await authMiddleware(req, res, () => {
      finished = true;
    });

    if (!finished) return;

    return handler(req as AuthenticatedRequest, res);
  };
}
