import crypto from 'crypto';
import type { NextApiRequest, NextApiResponse } from 'next';

type Data = {
  key?: string;
  error?: string;
}

export default function handler(req: NextApiRequest, res: NextApiResponse<Data>) {
  if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
  const { email, company, tier } = req.body || {};
  if (!email) return res.status(400).json({ error: 'Email is required' });

  // Simple demo key generator — replace with real issuance & persistence in production
  const key = crypto.randomBytes(24).toString('base64url');

  // TODO: persist key and metadata to a database and return associated info

  return res.status(200).json({ key });
}
