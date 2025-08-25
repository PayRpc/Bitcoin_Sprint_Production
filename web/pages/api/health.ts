import { performSystemHealthCheck } from "@/lib/maintenance";
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
    const health = await performSystemHealthCheck();
    
    // Set appropriate status code based on health
    const statusCode = health.status === "healthy" ? 200 : 
                      health.status === "maintenance" ? 503 : 207; // 207 Multi-Status for degraded

    return res.status(statusCode).json({
      ok: health.status === "healthy",
      service: 'web',
      ...health
    });
  } catch (e: any) {
    return res.status(500).json({
      ok: false,
      service: 'web',
      status: 'error',
      timestamp: new Date().toISOString(),
      error: e.message || "Failed to check system health",
    });
  }
}
