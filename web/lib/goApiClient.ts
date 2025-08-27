// Next.js API configuration to proxy to Go backend
const GO_API_BASE = process.env.GO_API_URL || 'http://localhost:8080';

export interface ApiResponse<T = any> {
  data?: T;
  error?: string;
  status: number;
}

export class GoApiClient {
  private baseUrl: string;
  private apiKey: string;

  constructor(baseUrl: string = GO_API_BASE, apiKey: string = process.env.API_KEY || '') {
    this.baseUrl = baseUrl;
    this.apiKey = apiKey;
  }

  private async request<T>(
    endpoint: string, 
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(this.apiKey && { 'X-API-Key': this.apiKey }),
      ...options.headers,
    };

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      const data = await response.json();
      
      return {
        data,
        status: response.status,
        ...(response.status >= 400 && { error: data.error || 'Request failed' }),
      };
    } catch (error) {
      return {
        status: 500,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }

  // Health check
  async health() {
    return this.request('/health');
  }

  // Status endpoint
  async status() {
    return this.request('/status');
  }

  // Latest data
  async latest() {
    return this.request('/latest');
  }

  // Metrics
  async metrics() {
    return this.request('/metrics');
  }

  // Key management
  async generateKey() {
    return this.request('/generate-key', { method: 'POST' });
  }

  async verifyKey(key: string) {
    return this.request('/verify-key', {
      method: 'POST',
      body: JSON.stringify({ key }),
    });
  }

  async renewKey() {
    return this.request('/renew', { method: 'POST' });
  }

  // Analytics
  async predictive() {
    return this.request('/predictive');
  }

  async adminMetrics() {
    return this.request('/admin-metrics');
  }

  async enterpriseAnalytics() {
    return this.request('/enterprise-analytics');
  }

  // V1 API endpoints
  async getLicenseInfo() {
    return this.request('/v1/license/info');
  }

  async getAnalyticsSummary() {
    return this.request('/v1/analytics/summary');
  }

  // Stream endpoint (WebSocket)
  createStream() {
    const wsUrl = this.baseUrl.replace('http', 'ws') + '/stream';
    return new WebSocket(wsUrl);
  }
}

export const goApiClient = new GoApiClient();
