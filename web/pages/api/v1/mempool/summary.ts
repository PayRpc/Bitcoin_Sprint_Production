import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../../../lib/auth';
import { withAuth } from '../../_withAuth';

/**
 * Mempool API endpoint that provides current mempool statistics
 * This is a mock implementation that would integrate with actual Bitcoin node in production
 */
async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  try {
    // In production this would make a call to the Bitcoin node
    // For now we're mocking the response with realistic mempool data
    const mempoolData = getMockMempoolData();
    
    // In enterprise tier we provide more detailed fee estimates
    const feeEstimates = req.apiKey.tier === 'ENTERPRISE' || req.apiKey.tier === 'ENTERPRISE_PLUS' 
      ? getEnterpriseDetailedFeeEstimates()
      : getBasicFeeEstimates();
    
    return res.status(200).json({
      mempool: mempoolData,
      feeEstimates,
      status: 'success',
      timestamp: new Date().toISOString()
    });
    
  } catch (error: any) {
    console.error('Error in mempool summary:', error);
    return res.status(500).json({
      error: 'Mempool data retrieval failed',
      message: error.message || 'Failed to retrieve mempool data',
      status: 'error'
    });
  }
}

/**
 * Mock mempool data generator
 * In production this would be replaced with actual Bitcoin node integration
 */
function getMockMempoolData() {
  // Create realistic mock data for demonstration
  const txCount = Math.floor(Math.random() * 20000) + 5000; // Random between 5k-25k transactions
  const size = txCount * 500; // Average tx size of 500 bytes
  const fees = (txCount * 0.00001).toFixed(8); // Average fee of 0.00001 BTC per tx
  
  // Fee rate brackets in satoshis per vbyte
  const feeRates = {
    '1-2': Math.floor(Math.random() * 1000) + 500,    // 1-2 sat/vB transactions
    '3-5': Math.floor(Math.random() * 2000) + 1000,   // 3-5 sat/vB transactions 
    '6-10': Math.floor(Math.random() * 3000) + 2000,  // 6-10 sat/vB transactions
    '11-20': Math.floor(Math.random() * 2000) + 1000, // 11-20 sat/vB transactions
    '21-50': Math.floor(Math.random() * 1000) + 500,  // 21-50 sat/vB transactions
    '51+': Math.floor(Math.random() * 500) + 100      // 51+ sat/vB transactions
  };
  
  return {
    txCount,
    size,
    bytes: size,
    usage: size,
    totalFees: fees,
    feeRateDistribution: feeRates,
    minFeeRate: 1,
    maxFeeRate: Math.floor(Math.random() * 100) + 50, // 50-150 sat/vB
    medianFeeRate: Math.floor(Math.random() * 10) + 5 // 5-15 sat/vB
  };
}

/**
 * Basic fee estimates for free/pro tiers
 */
function getBasicFeeEstimates() {
  return {
    fastestFee: Math.floor(Math.random() * 20) + 15, // 15-35 sat/vB
    halfHourFee: Math.floor(Math.random() * 10) + 10, // 10-20 sat/vB
    hourFee: Math.floor(Math.random() * 5) + 5, // 5-10 sat/vB
    economyFee: Math.floor(Math.random() * 3) + 2, // 2-5 sat/vB
    minimumFee: 1
  };
}

/**
 * Detailed fee estimates for enterprise tiers
 */
function getEnterpriseDetailedFeeEstimates() {
  return {
    fastestFee: Math.floor(Math.random() * 20) + 15, // 15-35 sat/vB
    halfHourFee: Math.floor(Math.random() * 10) + 10, // 10-20 sat/vB
    hourFee: Math.floor(Math.random() * 5) + 5, // 5-10 sat/vB
    economyFee: Math.floor(Math.random() * 3) + 2, // 2-5 sat/vB
    minimumFee: 1,
    // Detailed confirmation time estimates
    confirmedBlocks: {
      '2': Math.floor(Math.random() * 15) + 20, // fee for confirmation within 2 blocks
      '3': Math.floor(Math.random() * 10) + 15, // fee for confirmation within 3 blocks
      '6': Math.floor(Math.random() * 5) + 10,  // fee for confirmation within 6 blocks
      '12': Math.floor(Math.random() * 3) + 5,  // fee for confirmation within 12 blocks
      '24': Math.floor(Math.random() * 2) + 3,  // fee for confirmation within 24 blocks
    },
    confirmedMinutes: {
      '20': Math.floor(Math.random() * 15) + 20, // fee for confirmation within 20 minutes
      '40': Math.floor(Math.random() * 10) + 15, // fee for confirmation within 40 minutes
      '60': Math.floor(Math.random() * 5) + 10,  // fee for confirmation within 60 minutes
      '120': Math.floor(Math.random() * 3) + 5,  // fee for confirmation within 120 minutes
      '240': Math.floor(Math.random() * 2) + 3,  // fee for confirmation within 240 minutes
    },
    historicalRates: {
      '1h': Math.floor(Math.random() * 10) + 10,
      '6h': Math.floor(Math.random() * 15) + 5,
      '12h': Math.floor(Math.random() * 20) + 3,
      '24h': Math.floor(Math.random() * 25) + 2
    }
  };
}

export default withAuth(handler);
