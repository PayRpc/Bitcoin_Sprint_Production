import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '@/lib/auth';
import { withAuth } from '@/pages/api/_withAuth';

/**
 * Transaction lookup endpoint that provides information about a specific Bitcoin transaction
 * This is a mock implementation that would integrate with actual Bitcoin node in production
 */
async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { txid } = req.query;
  
  // Input validation
  if (!txid || typeof txid !== 'string') {
    return res.status(400).json({
      error: 'Invalid transaction ID',
      message: 'Please provide a valid Bitcoin transaction ID',
      status: 'error'
    });
  }
  
  // Validate that txid looks like a Bitcoin transaction ID (64 hex characters)
  const txidRegex = /^[0-9a-f]{64}$/i;
  if (!txidRegex.test(txid)) {
    return res.status(400).json({
      error: 'Invalid transaction ID format',
      message: 'Transaction ID must be a 64-character hexadecimal string',
      status: 'error'
    });
  }
  
  try {
    // In production this would make a call to the Bitcoin node
    // For now we're mocking the response with realistic transaction data
    const transactionData = getMockTransactionData(txid);
    
    // Track transaction requests in user metrics
    // await updateTransactionUsage(req.apiKey.id);
    
    return res.status(200).json({
      transaction: transactionData,
      status: 'success',
      timestamp: new Date().toISOString()
    });
    
  } catch (error: any) {
    console.error('Error in transaction lookup:', error);
    return res.status(500).json({
      error: 'Transaction lookup failed',
      message: error.message || 'Failed to retrieve transaction data',
      status: 'error'
    });
  }
}

/**
 * Mock transaction data generator
 * In production this would be replaced with actual Bitcoin node integration
 */
function getMockTransactionData(txid: string) {
  // Create realistic mock data for demonstration
  const now = Date.now();
  const minutesAgo = Math.floor(Math.random() * 120); // Random time in the last 2 hours
  const timestamp = Math.floor((now - (minutesAgo * 60 * 1000)) / 1000);
  
  return {
    txid: txid,
    hash: txid, // Same as txid for non-segwit transactions
    version: 2,
    size: Math.floor(Math.random() * 800) + 200, // Random size between 200-1000 bytes
    weight: Math.floor(Math.random() * 3200) + 800, // 4x size roughly
    locktime: 0,
    vin: [
      {
        txid: `${Math.random().toString(16).substring(2, 10)}${Math.random().toString(16).substring(2, 58)}`,
        vout: Math.floor(Math.random() * 3),
        scriptSig: {
          asm: "3045022100... [signature data]",
          hex: "483045022100..."
        },
        sequence: 4294967295
      }
    ],
    vout: [
      {
        value: (Math.random() * 1.5).toFixed(8), // Random BTC value
        n: 0,
        scriptPubKey: {
          asm: "OP_DUP OP_HASH160 [pubkey hash] OP_EQUALVERIFY OP_CHECKSIG",
          hex: "76a914...88ac",
          address: `bc1q${Math.random().toString(16).substring(2, 38)}`,
          type: "witness_v0_keyhash"
        }
      },
      {
        value: (Math.random() * 0.1).toFixed(8), // Change output
        n: 1,
        scriptPubKey: {
          asm: "OP_DUP OP_HASH160 [pubkey hash] OP_EQUALVERIFY OP_CHECKSIG",
          hex: "76a914...88ac",
          address: `bc1q${Math.random().toString(16).substring(2, 38)}`,
          type: "witness_v0_keyhash"
        }
      }
    ],
    hex: `0200000001${Math.random().toString(16).substring(2, 64)}...`, // Mock transaction hex
    blockhash: `00000000000000000${Math.random().toString(16).substring(2, 52)}`,
    confirmations: minutesAgo > 10 ? Math.floor(minutesAgo / 10) : 0, // Confirmations based on time
    time: timestamp,
    blocktime: timestamp,
    fee: (Math.random() * 0.0001).toFixed(8), // Random fee in BTC
    fee_sat: Math.floor(Math.random() * 10000) + 1000, // Fee in satoshis
    fee_per_vbyte: Math.floor(Math.random() * 20) + 5 // Fee rate
  };
}

export default withAuth(handler);
