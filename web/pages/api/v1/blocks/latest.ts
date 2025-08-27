import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '@/lib/auth';
import { withAuth } from '@/pages/api/_withAuth';

/**
 * Blocks endpoint that returns information about the latest blocks
 * This is a mock implementation that would integrate with actual Bitcoin node in production
 */
async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  try {
    // Parse query parameters
    const limit = parseInt(req.query.limit as string) || 10;
    const offset = parseInt(req.query.offset as string) || 0;
    
    // Validate parameters
    if (limit > 100) {
      return res.status(400).json({
        error: 'Invalid limit',
        message: 'Limit cannot exceed 100 blocks',
        status: 'error'
      });
    }
    
    // Generate mock block data
    const blocks = generateMockBlocks(limit, offset);
    
    // Return the response
    return res.status(200).json({
      blocks,
      count: blocks.length,
      offset,
      limit,
      status: 'success',
      timestamp: new Date().toISOString()
    });
    
  } catch (error: any) {
    console.error('Error in blocks endpoint:', error);
    return res.status(500).json({
      error: 'Failed to retrieve blocks',
      message: error.message || 'Internal server error',
      status: 'error'
    });
  }
}

/**
 * Generate mock block data for demonstration purposes
 * In production, this would be replaced with actual blockchain data
 */
function generateMockBlocks(limit: number, offset: number) {
  const blocks = [];
  const now = Date.now() / 1000; // Current time in seconds
  const blockTime = 600; // Average block time in seconds (10 minutes)
  const mockStartHeight = 850000; // A realistic recent block height
  
  for (let i = 0; i < limit; i++) {
    const blockHeight = mockStartHeight - offset - i;
    if (blockHeight < 0) break; // Don't go below genesis block
    
    const timestamp = Math.floor(now - ((offset + i) * blockTime));
    const txCount = Math.floor(Math.random() * 2000) + 1000; // Random tx count between 1000-3000
    const size = txCount * 250 + Math.floor(Math.random() * 50000); // Random block size
    const weight = size * 4; // Weight is roughly 4x size in Bitcoin
    
    blocks.push({
      height: blockHeight,
      hash: generateMockBlockHash(),
      timestamp: timestamp,
      txCount: txCount,
      size: size,
      weight: weight,
      confirmations: offset + i + 1
    });
  }
  
  return blocks;
}

/**
 * Generate a realistic-looking block hash (for demo purposes only)
 */
function generateMockBlockHash() {
  let hash = '00000000000000000';
  const chars = '0123456789abcdef';
  for (let i = 0; i < 47; i++) {
    hash += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return hash;
}

export default withAuth(handler);
