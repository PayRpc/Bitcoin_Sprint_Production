// Storage Verification API Client for Rust Web Server
const STORAGE_API_BASE = process.env.STORAGE_API_URL || 'http://localhost:8080';

export interface StorageVerificationRequest {
  file_id: string;
  provider: 'ipfs' | 'arweave' | 'filecoin' | 'bitcoin';
  protocol: string;
  file_size?: number;
}

export interface StorageVerificationResponse {
  verified: boolean;
  timestamp: number;
  signature: string;
  challenge_id: string;
  verification_score: number;
}

export interface StorageHealthResponse {
  status: string;
  timestamp: number;
  uptime_seconds: number;
}

export interface StorageMetricsResponse {
  active_challenges: number;
  total_verifications: number;
  rate_limited_requests: number;
  uptime_seconds: number;
  memory_usage_mb: number;
}

export class StorageApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = STORAGE_API_BASE) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(
    endpoint: string, 
    options: RequestInit = {}
  ): Promise<{ data?: T; error?: string; status: number }> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    try {
      const response = await fetch(url, {
        ...options,
        headers,
        // timeout: 10000, // 10 second timeout - not supported in fetch
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        return {
          status: response.status,
          error: errorData.error || `HTTP ${response.status}: ${response.statusText}`,
        };
      }

      const data = await response.json();
      
      return {
        data,
        status: response.status,
      };
    } catch (error) {
      return {
        status: 500,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }

  // Storage verification endpoints
  async verifyStorage(request: StorageVerificationRequest) {
    return this.request<StorageVerificationResponse>('/verify', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async health() {
    return this.request<StorageHealthResponse>('/health');
  }

  async metrics() {
    return this.request<StorageMetricsResponse>('/metrics');
  }

  // Utility method to check if storage server is available
  async isAvailable(): Promise<boolean> {
    try {
      const response = await this.health();
      return response.status === 200 && response.data?.status === 'healthy';
    } catch {
      return false;
    }
  }
}

export const storageApiClient = new StorageApiClient();
