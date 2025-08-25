import { getUpdateState } from "@/lib/updateState";
import type { NextApiRequest, NextApiResponse } from "next";

// ---------------------- API Handler ----------------------
export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  // CORS headers
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  if (req.method !== "GET") {
    return res.status(405).json({ ok: false, error: "Method not allowed" });
  }

  try {
    const state = await getUpdateState();
    return res.status(200).json({ ok: true, cached: true, ...state });
  } catch (e: any) {
    return res.status(500).json({
      ok: false,
      error: e.message || "Failed to read update state",
    });
  }
}
