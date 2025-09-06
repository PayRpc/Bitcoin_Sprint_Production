import type { NextApiRequest, NextApiResponse } from "next";
import { IncomingForm, Fields, Files } from 'formidable';
import fs from 'fs';
import path from 'path';
import crypto from 'crypto';

// IPFS Upload API for Bitcoin Sprint Enterprise
// Handles file uploads to IPFS with enterprise security and validation

// Configuration
const UPLOAD_DIR = path.join(process.cwd(), 'uploads');
const MAX_FILE_SIZE = 100 * 1024 * 1024; // 100MB from env config
const ALLOWED_TYPES = [
  'text/plain', 'application/json', 'image/jpeg', 'image/png', 
  'image/gif', 'application/pdf', 'text/csv', 'application/zip'
];

// Ensure upload directory exists
if (!fs.existsSync(UPLOAD_DIR)) {
  fs.mkdirSync(UPLOAD_DIR, { recursive: true });
}

export const config = {
  api: {
    bodyParser: false, // Required for formidable
  },
};

interface UploadResponse {
  success: boolean;
  cid?: string;
  hash?: string;
  size?: number;
  filename?: string;
  message?: string;
  error?: string;
}

// Mock IPFS client - replace with actual IPFS integration
class MockIPFSClient {
  async add(filePath: string): Promise<{ cid: string; size: number }> {
    // Generate mock CID based on file hash
    const fileBuffer = fs.readFileSync(filePath);
    const hash = crypto.createHash('sha256').update(fileBuffer).digest('hex');
    const cid = `Qm${hash.substring(0, 44)}`; // Mock IPFS CID format
    
    // In production, replace with actual IPFS client:
    // const ipfs = create({ url: 'http://localhost:5001' });
    // const result = await ipfs.add(fs.createReadStream(filePath));
    // return { cid: result.cid.toString(), size: result.size };
    
    return { cid, size: fileBuffer.length };
  }

  async pin(cid: string): Promise<void> {
    // Mock pinning - replace with actual IPFS pinning
    console.log(`[IPFS] Pinning ${cid} (mock)`);
  }
}

const ipfsClient = new MockIPFSClient();

export default async function handler(req: NextApiRequest, res: NextApiResponse<UploadResponse>) {
  // CORS headers
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "POST, OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization");

  if (req.method === "OPTIONS") {
    return res.status(200).end();
  }

  if (req.method !== "POST") {
    return res.status(405).json({ 
      success: false, 
      error: "Method not allowed. Use POST to upload files." 
    });
  }

  try {
    // Parse form data
    const form = new IncomingForm({
      uploadDir: UPLOAD_DIR,
      keepExtensions: true,
      maxFileSize: MAX_FILE_SIZE,
      maxFiles: 1
    });

    const [fields, files] = await new Promise<[Fields, Files]>((resolve, reject) => {
      form.parse(req, (err, fields, files) => {
        if (err) reject(err);
        else resolve([fields, files]);
      });
    });

    // Validate file upload
    const uploadedFile = Array.isArray(files.file) ? files.file[0] : files.file;
    if (!uploadedFile) {
      return res.status(400).json({
        success: false,
        error: "No file uploaded"
      });
    }

    // Validate file type
    if (!ALLOWED_TYPES.includes(uploadedFile.mimetype || '')) {
      // Clean up uploaded file
      fs.unlinkSync(uploadedFile.filepath);
      return res.status(400).json({
        success: false,
        error: `File type not allowed. Supported types: ${ALLOWED_TYPES.join(', ')}`
      });
    }

    // Validate file size
    if (uploadedFile.size > MAX_FILE_SIZE) {
      fs.unlinkSync(uploadedFile.filepath);
      return res.status(400).json({
        success: false,
        error: `File too large. Maximum size: ${MAX_FILE_SIZE / 1024 / 1024}MB`
      });
    }

    // Upload to IPFS
    console.log(`[IPFS UPLOAD] Processing file: ${uploadedFile.originalFilename} (${uploadedFile.size} bytes)`);
    const ipfsResult = await ipfsClient.add(uploadedFile.filepath);
    
    // Pin the file
    await ipfsClient.pin(ipfsResult.cid);

    // Generate file hash for verification
    const fileBuffer = fs.readFileSync(uploadedFile.filepath);
    const fileHash = crypto.createHash('sha256').update(fileBuffer).digest('hex');

    // Clean up temporary file
    fs.unlinkSync(uploadedFile.filepath);

    console.log(`[IPFS UPLOAD] Success: ${ipfsResult.cid}`);

    return res.status(200).json({
      success: true,
      cid: ipfsResult.cid,
      hash: fileHash,
      size: ipfsResult.size,
      filename: uploadedFile.originalFilename || 'unknown',
      message: `File uploaded to IPFS successfully. CID: ${ipfsResult.cid}`
    });

  } catch (error) {
    console.error('[IPFS UPLOAD] Error:', error);
    
    return res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : 'Upload failed'
    });
  }
}
