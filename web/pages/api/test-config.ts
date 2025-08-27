// API endpoint to test Bitcoin Sprint configuration
import type { NextApiRequest, NextApiResponse } from 'next';

interface ConfigTestRequest {
  license_key: string;
  rpc_nodes: string[];
  rpc_user: string;
  rpc_pass: string;
  turbo_mode: boolean;
}

interface ConfigTestResponse {
  valid: boolean;
  license_status: string;
  rpc_connectivity: string;
  turbo_enabled: boolean;
  recommendations?: string[];
  errors?: string[];
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<ConfigTestResponse>
) {
  if (req.method !== 'POST') {
    return res.status(405).json({
      valid: false,
      license_status: 'invalid',
      rpc_connectivity: 'untested',
      turbo_enabled: false,
      errors: ['Method not allowed']
    });
  }

  try {
    const config: ConfigTestRequest = req.body;
    
    console.log('[CONFIG TEST] Testing configuration:', {
      license_key: config.license_key?.substring(0, 20) + '...',
      rpc_nodes: config.rpc_nodes,
      rpc_user: config.rpc_user,
      turbo_mode: config.turbo_mode
    });

    const recommendations: string[] = [];
    const errors: string[] = [];

    // Validate license key format
    let license_status = 'invalid';
    if (config.license_key) {
      if (config.license_key.startsWith('sprint-entplus_')) {
        license_status = 'valid_enterprise_plus';
      } else if (config.license_key.startsWith('sprint-ent_')) {
        license_status = 'valid_enterprise';
      } else if (config.license_key.startsWith('sprint-turbo_')) {
        license_status = 'valid_turbo';
      } else if (config.license_key.startsWith('sprint-free_')) {
        license_status = 'valid_free';
      } else {
        license_status = 'invalid_format';
        errors.push('Invalid license key format');
      }
    } else {
      errors.push('License key is required');
    }

    // Test RPC connectivity (mock for now)
    let rpc_connectivity = 'unknown';
    if (config.rpc_nodes && config.rpc_nodes.length > 0) {
      // Check if using localhost
      const hasLocalhost = config.rpc_nodes.some(node => 
        node.includes('localhost') || node.includes('127.0.0.1')
      );
      
      if (hasLocalhost) {
        rpc_connectivity = 'localhost_detected';
        recommendations.push('Using localhost RPC - ensure Bitcoin Core is running');
      } else {
        rpc_connectivity = 'external_nodes';
        recommendations.push('Using external RPC nodes - good for production');
      }
    } else {
      rpc_connectivity = 'no_nodes';
      errors.push('At least one RPC node is required');
    }

    // Validate RPC credentials
    if (!config.rpc_user || config.rpc_user === 'your-rpc-user') {
      errors.push('Please set a valid RPC username');
    }
    if (!config.rpc_pass || config.rpc_pass === 'your-rpc-password') {
      errors.push('Please set a valid RPC password');
    }

    // Turbo mode validation
    const turbo_enabled = config.turbo_mode === true;
    if (turbo_enabled && !license_status.includes('enterprise') && !license_status.includes('turbo')) {
      recommendations.push('Turbo mode requires Enterprise or Turbo license');
    }

    // Additional recommendations
    if (license_status.includes('valid')) {
      recommendations.push('License key format is valid');
    }
    
    if (config.rpc_nodes?.length > 1) {
      recommendations.push('Multiple RPC nodes configured for failover');
    }

    const isValid = errors.length === 0 && license_status.includes('valid');

    const response: ConfigTestResponse = {
      valid: isValid,
      license_status,
      rpc_connectivity,
      turbo_enabled,
      recommendations: recommendations.length > 0 ? recommendations : undefined,
      errors: errors.length > 0 ? errors : undefined
    };

    console.log('[CONFIG TEST] Result:', response);

    return res.status(200).json(response);

  } catch (error) {
    console.error('[CONFIG TEST] Error:', error);
    return res.status(500).json({
      valid: false,
      license_status: 'error',
      rpc_connectivity: 'error',
      turbo_enabled: false,
      errors: ['Internal server error']
    });
  }
}
