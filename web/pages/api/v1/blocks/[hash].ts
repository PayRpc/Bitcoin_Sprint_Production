import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '@/lib/auth';
import { withAuth } from '@/pages/api/_withAuth';

/**
 * Block details endpoint that provides information about a specific block by hash
 * This is a mock implementation that would integrate with actual Bitcoin node in production
 */
async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { hash } = req.query;
  
  // Input validation
  if (!hash || typeof hash !== 'string') {
    return res.status(400).json({
      error: 'Invalid block hash',
      message: 'Please provide a valid Bitcoin block hash',
      status: 'error'
    });
  }
  
  // Validate that hash looks like a Bitcoin block hash
  const hashRegex = /^[0-9a-f]{64}$/i;
  if (!hashRegex.test(hash)) {
    return res.status(400).json({
      error: 'Invalid block hash format',
      message: 'Block hash must be a 64-character hexadecimal string',
      status: 'error'
    });
  }
  
  try {
    // In production this would make a call to the Bitcoin node
    // For now we're mocking the response with realistic block data
    const blockData = getMockBlockData(hash);
    
    return res.status(200).json({
      block: blockData,
      status: 'success',
      timestamp: new Date().toISOString()
    });
    
  } catch (error: any) {
    console.error('Error in block lookup:', error);
    return res.status(500).json({
      error: 'Block lookup failed',
      message: error.message || 'Failed to retrieve block data',
      status: 'error'
    });
  }
}

/**
 * Mock block data generator
 * In production this would be replaced with actual Bitcoin node integration
 */
function getMockBlockData(hash: string) {
  // Create realistic mock data for demonstration
  const now = Date.now() / 1000;
  const minutesAgo = Math.floor(Math.random() * 120); // Random time in the last 2 hours
  const timestamp = Math.floor(now - (minutesAgo * 60));
  const height = 850000 - Math.floor(minutesAgo / 10); // Approximate height based on time
  const txCount = Math.floor(Math.random() * 2000) + 1000; // Random tx count between 1000-3000
  const size = txCount * 250 + Math.floor(Math.random() * 50000); // Random block size
  
  // Generate some mock transaction IDs
  const transactions = [];
  for (let i = 0; i < 10; i++) { // Just show 10 txs for simplicity
    transactions.push(generateMockTxid());
  }
  
  return {
    hash: hash,
    confirmations: Math.floor(minutesAgo / 10) + 1,
    size: size,
    weight: size * 4,
    height: height,
    version: 0x20000000,
    versionHex: "20000000",
    merkleroot: generateMockTxid(),
    time: timestamp,
    mediantime: timestamp - 300,
    nonce: Math.floor(Math.random() * 4294967296),
    bits: "1703a5b3",
    difficulty: 53311599263588.1,
    chainwork: "00000000000000000000000000000000000000001ec9886c4754f9a8e8778a6",
    nTx: txCount,
    previousblockhash: generateMockBlockHash(),
    nextblockhash: height < 850000 ? generateMockBlockHash() : null,
    strippedsize: Math.floor(size * 0.8),
    transactions: transactions,
    transactionCount: txCount,
    totalFees: (0.25 + (Math.random() * 0.5)).toFixed(8),
    miner: "Unknown",
    poolName: pickRandomPool(),
    avgFeeRate: Math.floor(Math.random() * 15) + 5,
    avgFeePerTx: Math.floor(Math.random() * 20000) + 10000
  };
}

/**
 * Generate a realistic-looking block hash
 */
function generateMockBlockHash() {
  let hash = '00000000000000000';
  const chars = '0123456789abcdef';
  for (let i = 0; i < 47; i++) {
    hash += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return hash;
}

/**
 * Generate a realistic-looking transaction ID
 */
function generateMockTxid() {
  let txid = '';
  const chars = '0123456789abcdef';
  for (let i = 0; i < 64; i++) {
    txid += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return txid;
}

/**
 * Pick a random mining pool name
 */
function pickRandomPool() {
  const pools = [
    "Foundry USA",
    "AntPool",
    "F2Pool",
    "Binance Pool",
    "ViaBTC",
    "Poolin",
    "SlushPool",
    "BTC.com",
    "Unknown"
  ];
  return pools[Math.floor(Math.random() * pools.length)];
}

export default withAuth(handler);
