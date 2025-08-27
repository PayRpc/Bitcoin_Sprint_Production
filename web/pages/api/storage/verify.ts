import type { NextApiRequest, NextApiResponse } from "next";
import { storageApiClient, StorageVerificationRequest } from "../../../lib/storageApiClient";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  // CORS headers
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization");

  if (req.method === "OPTIONS") {
    return res.status(200).end();
  }

  if (req.method !== "POST") {
    return res.status(405).json({ 
      ok: false, 
      error: "Method not allowed. Use POST to verify storage." 
    });
  }

  try {
    // Validate request body
    const { file_id, provider, protocol, file_size } = req.body as StorageVerificationRequest;

    if (!file_id || !provider || !protocol) {
      return res.status(400).json({
        ok: false,
        error: "Missing required fields: file_id, provider, protocol"
      });
    }

    // Validate provider
    const validProviders = ['ipfs', 'arweave', 'filecoin', 'bitcoin'];
    if (!validProviders.includes(provider)) {
      return res.status(400).json({
        ok: false,
        error: `Invalid provider. Must be one of: ${validProviders.join(', ')}`
      });
    }

    // Validate file_size if provided
    if (file_size !== undefined && (file_size < 0 || file_size > 10 * 1024 * 1024 * 1024)) { // Max 10GB
      return res.status(400).json({
        ok: false,
        error: "Invalid file_size. Must be between 0 and 10GB"
      });
    }

    // Check if storage server is available
    const isStorageAvailable = await storageApiClient.isAvailable();
    if (!isStorageAvailable) {
      return res.status(503).json({
        ok: false,
        error: "Storage verification service is currently unavailable"
      });
    }

    // Make verification request to Rust storage server
    const verificationResponse = await storageApiClient.verifyStorage({
      file_id,
      provider,
      protocol,
      file_size: file_size || 1024 * 1024 // Default 1MB
    });

    if (verificationResponse.error) {
      return res.status(verificationResponse.status).json({
        ok: false,
        error: verificationResponse.error
      });
    }

    // Return successful verification response
    return res.status(200).json({
      ok: true,
      verification: verificationResponse.data,
      timestamp: Date.now(),
      service: 'bitcoin-sprint-storage'
    });

  } catch (error) {
    console.error('Storage verification error:', error);
    return res.status(500).json({
      ok: false,
      error: 'Internal server error during storage verification'
    });
  }
}
