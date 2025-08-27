import { goApiClient } from "../../lib/goApiClient";
import type { NextApiRequest, NextApiResponse } from "next";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  // CORS headers
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  if (req.method !== "GET") {
    return res.status(405).json({ ok: false, error: "Method not allowed" });
  }

  try {
    const response = await goApiClient.health();
    
    if (response.error) {
      return res.status(response.status).json({
        ok: false,
        service: 'bitcoin-sprint',
        error: response.error,
        status: 'unhealthy'
      });
    }

    return res.status(200).json({
      ok: true,
      service: 'bitcoin-sprint',
      ...(response.data || {})
    });
  } catch (error: any) {
    return res.status(500).json({
      ok: false,
      service: 'bitcoin-sprint',
      status: 'error',
      error: error.message || 'Health check failed'
    });
  }
}
