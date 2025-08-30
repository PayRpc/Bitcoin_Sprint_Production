// TypeScript wrapper for Rust Entropy Bridge
// This provides type-safe access to the entropy bridge functionality

export interface EntropyBridgeStatus {
  available: boolean;
  rustAvailable: boolean;
  fallbackMode: boolean;
}

export interface EntropyBridge {
  generateAdminSecret(encoding?: 'hex' | 'base64' | 'raw'): Promise<string>;
  isAvailable(): boolean;
  getStatus(): EntropyBridgeStatus;
}

// Dynamic import for the JavaScript entropy bridge
let entropyBridge: EntropyBridge | null = null;
let bridgePromise: Promise<EntropyBridge> | null = null;

async function loadEntropyBridge(): Promise<EntropyBridge> {
  if (entropyBridge) {
    return entropyBridge;
  }

  if (bridgePromise) {
    return bridgePromise;
  }

  bridgePromise = (async () => {
    try {
      // Import the JavaScript entropy bridge
      const bridgeModule = await import('../rust-entropy-bridge.js') as any;
      entropyBridge = bridgeModule.getEntropyBridge();
      if (!entropyBridge) {
        throw new Error('Entropy bridge initialization failed');
      }
      return entropyBridge;
    } catch (error) {
      console.error('Failed to load entropy bridge:', error);
      throw new Error('Entropy bridge not available');
    }
  })();

  return bridgePromise;
}

export async function getEntropyBridge(): Promise<EntropyBridge> {
  return loadEntropyBridge();
}

export async function generateAdminSecret(encoding: 'hex' | 'base64' | 'raw' = 'base64'): Promise<string> {
  const bridge = await getEntropyBridge();
  return bridge.generateAdminSecret(encoding);
}

export async function isEntropyBridgeAvailable(): Promise<boolean> {
  try {
    const bridge = await getEntropyBridge();
    return bridge.isAvailable();
  } catch {
    return false;
  }
}

export async function getEntropyStatus(): Promise<EntropyBridgeStatus> {
  try {
    const bridge = await getEntropyBridge();
    return bridge.getStatus();
  } catch {
    return {
      available: false,
      rustAvailable: false,
      fallbackMode: true
    };
  }
}
