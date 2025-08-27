import { storageApiClient } from "../../../lib/storageApiClient";
import type { NextApiRequest, NextApiResponse } from "next";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  // CORS headers
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  if (req.method !== "GET") {
    return res.status(405).json({ 
      ok: false, 
      error: "Method not allowed" 
    });
  }

  try {
    const response = await storageApiClient.metrics();
    
    if (response.error) {
      return res.status(response.status).json({
        ok: false,
        service: 'bitcoin-sprint-storage',
        error: response.error
      });
    }

    return res.status(200).json({
      ok: true,
      service: 'bitcoin-sprint-storage',
      metrics: response.data,
      timestamp: Date.now()
    });

  } catch (error) {
    console.error('Storage metrics error:', error);
    return res.status(500).json({
      ok: false,
      service: 'bitcoin-sprint-storage',
      error: 'Failed to retrieve metrics'
    });
  }
}
