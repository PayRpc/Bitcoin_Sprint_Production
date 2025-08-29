import { readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

// Dynamic import for FFI (ESM compatible)
let ffi: any = null;
let ref: any = null;

try {
  // Dynamic imports for ESM compatibility
  const ffiModule = await import('ffi-napi');
  const refModule = await import('ref-napi');
  ffi = ffiModule.default;
  ref = refModule.default;
} catch (error) {
  console.warn('FFI modules not available, using fallback mode');
}

/**
 * Rust Entropy Bridge for Admin Authentication
 * Provides secure admin secret generation using enterprise entropy
 */
export class RustEntropyBridge {
  private lib: any = null;
  private isInitialized = false;

  constructor() {
    this.initialize();
  }

  private async initialize() {
    if (!ffi || !ref) {
      console.warn('FFI not available, using fallback entropy generation');
      this.isInitialized = false;
      return;
    }

    try {
      // Determine library path based on platform
      const __filename = fileURLToPath(import.meta.url);
      const __dirname = dirname(__filename);
      const libPath = this.getLibraryPath(__dirname);

      // Define function signatures
      const libDefinition = {
        // Generate admin secret as raw bytes
        'generate_admin_secret_c': [ref.types.int, [ref.refType(ref.types.uint8), ref.types.size_t]],

        // Generate admin secret as base64 string
        'generate_admin_secret_base64_c': [ref.types.int, [ref.refType(ref.types.char), ref.types.size_t]],

        // Generate admin secret as hex string
        'generate_admin_secret_hex_c': [ref.types.int, [ref.refType(ref.types.char), ref.types.size_t]],
      };

      // Load the library
      this.lib = ffi.Library(libPath, libDefinition);
      this.isInitialized = true;
      console.log('✅ Rust entropy bridge initialized successfully');
    } catch (error) {
      console.warn('Failed to initialize Rust entropy bridge:', error);
      this.isInitialized = false;
    }
  }

  private getLibraryPath(baseDir: string): string {
    const platform = process.platform;
    const arch = process.arch;

    if (platform === 'win32') {
      if (arch === 'x64') {
        return join(baseDir, '..', '..', '..', 'secure', 'rust', 'target', 'release', 'securebuffer.dll');
      }
    } else if (platform === 'linux') {
      if (arch === 'x64') {
        return join(baseDir, '..', '..', '..', 'secure', 'rust', 'target', 'release', 'libsecurebuffer.so');
      }
    } else if (platform === 'darwin') {
      if (arch === 'x64' || arch === 'arm64') {
        return join(baseDir, '..', '..', '..', 'secure', 'rust', 'target', 'release', 'libsecurebuffer.dylib');
      }
    }

    throw new Error(`Unsupported platform: ${platform} ${arch}`);
  }

  /**
   * Generate admin secret using enterprise entropy
   * @param encoding - Output encoding ('raw', 'base64', 'hex')
   * @returns Promise<string> - The generated admin secret
   */
  async generateAdminSecret(encoding: 'raw' | 'base64' | 'hex' = 'base64'): Promise<string> {
    if (!this.isInitialized) {
      return this.fallbackGenerateSecret(encoding);
    }

    try {
      switch (encoding) {
        case 'raw':
          return await this.generateRawSecret();
        case 'base64':
          return await this.generateBase64Secret();
        case 'hex':
          return await this.generateHexSecret();
        default:
          throw new Error(`Unsupported encoding: ${encoding}`);
      }
    } catch (error) {
      console.warn('Rust entropy generation failed, using fallback:', error);
      return this.fallbackGenerateSecret(encoding);
    }
  }

  private async generateRawSecret(): Promise<string> {
    const outputBuffer = Buffer.alloc(32);
    const result = this.lib.generate_admin_secret_c(outputBuffer, outputBuffer.length);

    if (result !== 0) {
      throw new Error(`Rust function returned error code: ${result}`);
    }

    // Return as hex string for storage/transport
    return outputBuffer.toString('hex');
  }

  private async generateBase64Secret(): Promise<string> {
    const outputBuffer = Buffer.alloc(256); // Large enough for base64 + null terminator
    const result = this.lib.generate_admin_secret_base64_c(outputBuffer, outputBuffer.length);

    if (result !== 0) {
      throw new Error(`Rust function returned error code: ${result}`);
    }

    // Convert to string and trim null terminator
    return outputBuffer.toString('utf8').replace(/\0/g, '');
  }

  private async generateHexSecret(): Promise<string> {
    const outputBuffer = Buffer.alloc(256); // Large enough for hex + null terminator
    const result = this.lib.generate_admin_secret_hex_c(outputBuffer, outputBuffer.length);

    if (result !== 0) {
      throw new Error(`Rust function returned error code: ${result}`);
    }

    // Convert to string and trim null terminator
    return outputBuffer.toString('utf8').replace(/\0/g, '');
  }

  /**
   * Fallback entropy generation when Rust is not available
   */
  private fallbackGenerateSecret(encoding: 'raw' | 'base64' | 'hex'): string {
    const crypto = await import('crypto');

    // Generate 32 bytes of entropy
    const entropy = crypto.randomBytes(32);

    switch (encoding) {
      case 'raw':
        return entropy.toString('hex');
      case 'base64':
        return entropy.toString('base64');
      case 'hex':
        return entropy.toString('hex');
      default:
        throw new Error(`Unsupported encoding: ${encoding}`);
    }
  }

  /**
   * Check if the Rust bridge is available
   */
  isAvailable(): boolean {
    return this.isInitialized;
  }

  /**
   * Get bridge status information
   */
  getStatus(): { available: boolean; rustAvailable: boolean; fallbackMode: boolean } {
    return {
      available: true,
      rustAvailable: this.isInitialized,
      fallbackMode: !this.isInitialized,
    };
  }
}

// Singleton instance
let entropyBridge: RustEntropyBridge | null = null;

/**
 * Get the global entropy bridge instance
 */
export function getEntropyBridge(): RustEntropyBridge {
  if (!entropyBridge) {
    entropyBridge = new RustEntropyBridge();
  }
  return entropyBridge;
}

/**
 * Generate admin secret using enterprise entropy
 * @param encoding - Output encoding ('raw', 'base64', 'hex')
 * @returns Promise<string> - The generated admin secret
 */
export async function generateAdminSecret(encoding: 'raw' | 'base64' | 'hex' = 'base64'): Promise<string> {
  const bridge = getEntropyBridge();
  return bridge.generateAdminSecret(encoding);
}

/**
 * Check if Rust entropy bridge is available
 */
export function isEntropyBridgeAvailable(): boolean {
  const bridge = getEntropyBridge();
  return bridge.isAvailable();
}
