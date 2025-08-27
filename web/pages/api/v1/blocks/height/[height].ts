import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '@/lib/auth';
import { withAuth } from '@/pages/api/_withAuth';

/**
 * Endpoint for getting height-specific block information
 * This is a mock implementation that would integrate with actual Bitcoin node in production
 */
async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { height } = req.query;
  
  // Input validation
  if (!height || typeof height !== 'string') {
    return res.status(400).json({
      error: 'Invalid block height',
      message: 'Please provide a valid Bitcoin block height',
      status: 'error'
    });
  }
  
  // Parse height and validate
  const blockHeight = parseInt(height);
  if (isNaN(blockHeight) || blockHeight < 0) {
    return res.status(400).json({
      error: 'Invalid block height format',
      message: 'Block height must be a non-negative integer',
      status: 'error'
    });
  }
  
  try {
    // In production this would make a call to the Bitcoin node
    // For now we're mocking the response with realistic block data
    const blockData = getMockBlockDataByHeight(blockHeight);
    
    return res.status(200).json({
      block: blockData,
      status: 'success',
      timestamp: new Date().toISOString()
    });
    
  } catch (error: any) {
    console.error('Error in block lookup by height:', error);
    return res.status(500).json({
      error: 'Block lookup failed',
      message: error.message || 'Failed to retrieve block data',
      status: 'error'
    });
  }
}

/**
 * Mock block data generator based on height
 * In production this would be replaced with actual Bitcoin node integration
 */
function getMockBlockDataByHeight(height: number) {
  // Create realistic mock data for demonstration
  const mockLatestHeight = 850000;
  
  // If requested height is greater than our mock latest, return a 404
  if (height > mockLatestHeight) {
    throw new Error('Block not found');
  }
  
  const blocksBack = mockLatestHeight - height;
  const now = Date.now() / 1000;
  const timestamp = Math.floor(now - (blocksBack * 600)); // 10 minutes per block
  const txCount = Math.floor(Math.random() * 2000) + 1000; // Random tx count between 1000-3000
  const size = txCount * 250 + Math.floor(Math.random() * 50000); // Random block size
  
  // Generate a deterministic hash based on height for consistency
  const hash = generateMockBlockHashFromHeight(height);
  
  // Generate some mock transaction IDs
  const transactions = [];
  for (let i = 0; i < 10; i++) { // Just show 10 txs for simplicity
    transactions.push(generateMockTxid());
  }
  
  return {
    hash: hash,
    confirmations: blocksBack + 1,
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
    previousblockhash: height > 0 ? generateMockBlockHashFromHeight(height - 1) : "0000000000000000000000000000000000000000000000000000000000000000",
    nextblockhash: height < mockLatestHeight ? generateMockBlockHashFromHeight(height + 1) : null,
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
 * Generate a realistic-looking block hash from height (deterministic)
 */
function generateMockBlockHashFromHeight(height: number) {
  // Use the height to seed the "random" generation for consistent results
  const seed = height.toString();
  let hash = '00000000000000000';
  const chars = '0123456789abcdef';
  
  for (let i = 0; i < 47; i++) {
    // Simple deterministic hash function based on height and position
    const charIndex = (height + i * 7) % 16;
    hash += chars.charAt(charIndex);
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
