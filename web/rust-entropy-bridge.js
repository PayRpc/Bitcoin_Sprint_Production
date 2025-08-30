import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
// Dynamic import for FFI (ESM compatible)
let ffi = null;
let ref = null;
let ffiAvailable = false;
let ffiInitialized = false;

async function initializeFFI() {
    if (ffiInitialized) return;
    ffiInitialized = true;
    
    try {
        // Use eval to prevent webpack from resolving these at build time
        const importFFI = new Function('return import("ffi-napi")');
        const importRef = new Function('return import("ref-napi")');
        
        const ffiModule = await importFFI();
        const refModule = await importRef();
        ffi = ffiModule.default;
        ref = refModule.default;
        ffiAvailable = true;
    }
    catch (error) {
        console.warn('FFI modules not available, using Node.js crypto fallback');
        console.warn('To enable Rust entropy: npm install ffi-napi ref-napi ref-struct-di');
        ffiAvailable = false;
    }
}
/**
 * Rust Entropy Bridge for Admin Authentication
 * Provides secure admin secret generation using enterprise entropy
 */
export class RustEntropyBridge {
    lib = null;
    isInitialized = false;
    initPromise = null;
    
    constructor() {
        // Don't initialize automatically - wait for explicit call
    }
    
    async initialize() {
        if (this.initPromise) {
            return this.initPromise;
        }
        
        this.initPromise = (async () => {
            await initializeFFI();
            this.isInitialized = ffiAvailable;
            if (ffiAvailable) {
                this.initializeFFI();
            }
            else {
                console.log('✅ Rust entropy bridge initialized with Node.js crypto fallback');
            }
        })();
        
        return this.initPromise;
    }
    
    initializeFFI() {
        if (!ffiAvailable || !ffi || !ref) {
            console.warn('FFI not available during initialization');
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
            console.log('✅ Rust entropy bridge initialized successfully with FFI');
        }
        catch (error) {
            console.warn('Failed to initialize Rust entropy bridge FFI:', error);
            console.warn('Falling back to Node.js crypto');
            this.isInitialized = false;
        }
    }
    getLibraryPath(baseDir) {
        const platform = process.platform;
        const arch = process.arch;
        if (platform === 'win32') {
            if (arch === 'x64') {
                return join(baseDir, '..', '..', '..', 'secure', 'rust', 'target', 'release', 'securebuffer.dll');
            }
        }
        else if (platform === 'linux') {
            if (arch === 'x64') {
                return join(baseDir, '..', '..', '..', 'secure', 'rust', 'target', 'release', 'libsecurebuffer.so');
            }
        }
        else if (platform === 'darwin') {
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
    async generateAdminSecret(encoding = 'base64') {
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
        }
        catch (error) {
            console.warn('Rust entropy generation failed, using fallback:', error);
            return this.fallbackGenerateSecret(encoding);
        }
    }
    async generateRawSecret() {
        const outputBuffer = Buffer.alloc(32);
        const result = this.lib.generate_admin_secret_c(outputBuffer, outputBuffer.length);
        if (result !== 0) {
            throw new Error(`Rust function returned error code: ${result}`);
        }
        // Return as hex string for storage/transport
        return outputBuffer.toString('hex');
    }
    async generateBase64Secret() {
        const outputBuffer = Buffer.alloc(256); // Large enough for base64 + null terminator
        const result = this.lib.generate_admin_secret_base64_c(outputBuffer, outputBuffer.length);
        if (result !== 0) {
            throw new Error(`Rust function returned error code: ${result}`);
        }
        // Convert to string and trim null terminator
        return outputBuffer.toString('utf8').replace(/\0/g, '');
    }
    async generateHexSecret() {
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
    async fallbackGenerateSecret(encoding) {
        // Use Node.js crypto for fallback
        const crypto = await import('crypto');
        // Generate 32 bytes of entropy
        const entropy = crypto.default.randomBytes(32);
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
    isAvailable() {
        return this.isInitialized;
    }
    /**
     * Get bridge status information
     */
    getStatus() {
        return {
            available: true,
            rustAvailable: this.isInitialized && ffiAvailable,
            fallbackMode: !this.isInitialized || !ffiAvailable,
        };
    }
}
// Singleton instance
let entropyBridge = null;
/**
 * Get the global entropy bridge instance
 */
export async function getEntropyBridge() {
    if (!entropyBridge) {
        entropyBridge = new RustEntropyBridge();
        await entropyBridge.initialize();
    }
    return entropyBridge;
}
/**
 * Generate admin secret using enterprise entropy
 * @param encoding - Output encoding ('raw', 'base64', 'hex')
 * @returns Promise<string> - The generated admin secret
 */
export async function generateAdminSecret(encoding = 'base64') {
    const bridge = await getEntropyBridge();
    return bridge.generateAdminSecret(encoding);
}
/**
 * Check if Rust entropy bridge is available
 */
export async function isEntropyBridgeAvailable() {
    const bridge = await getEntropyBridge();
    return bridge.isAvailable();
}
