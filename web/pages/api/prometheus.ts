import { register } from "@/lib/prometheus";
import type { NextApiRequest, NextApiResponse } from "next";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method !== "GET") {
    return res.status(405).json({ error: "Method not allowed" });
  }

  try {
    // Set Prometheus content type
    res.setHeader('Content-Type', register.contentType);
    
    // Get metrics from Prometheus registry
    const metrics = await register.metrics();
    
    res.status(200).send(metrics);
  } catch (error: any) {
    console.error('Error generating Prometheus metrics:', error);
    res.status(500).json({ 
      error: "Failed to generate metrics",
      details: error.message 
    });
  }
}
