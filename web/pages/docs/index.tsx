import Head from "next/head";
import Image from "next/image";
import ConfigSnippet from "../../components/ConfigSnippet";
import Badge from "../../components/ui/badge";
import { Card, CardContent } from "../../components/ui/card";
import CopyButton from "../../components/ui/copyButton";
import pkg from "../../package.json";

const version = pkg.version || "1.0.0";

export default function DocsPage() {
  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-950 to-[#0a0a0a] text-gray-100">
      <Head>
        <title>Bitcoin Sprint - Docs</title>
        <meta name="description" content="Documentation and configuration for Bitcoin Sprint. Keep your RPC credentials on your own server; the relay uses an API key." />
        <meta property="og:title" content="Bitcoin Sprint Docs" />
        <meta property="og:description" content="How to configure Bitcoin Sprint and where to store RPC credentials." />
      </Head>

      <main className="max-w-5xl mx-auto py-16 px-6">
        <header className="text-center mb-10">
          <div className="flex items-center justify-center space-x-4">
            <Image src="/logo-bitcoin-sprint.svg" alt="Logo" width={96} height={96} />
            <div className="text-left">
              <h1 className="text-3xl font-extrabold text-white">Bitcoin Sprint — Documentation</h1>
              <div className="flex items-center space-x-3 mt-1">
                <span className="text-sm text-gray-300">v{version}</span>
                <Badge>Stable</Badge>
                <Badge color="orange">Recommended</Badge>
              </div>
              <p className="text-gray-400 mt-2">Secure, fast relay for your Bitcoin node. Keep RPC credentials on your server.</p>
            </div>
          </div>
        </header>

        <section className="grid grid-cols-1 gap-6 mb-8">
          <Card className="bg-gray-850 border-gray-800">
            <CardContent>
              <h2 className="text-2xl font-semibold text-white mb-2">Quick Start</h2>
              <p className="text-gray-300">Drop a <code className="bg-gray-900 px-1 rounded">config.json</code> on your server, set your RPC credentials, and launch the relay. The relay authenticates requests using a per-license API key. See examples below.</p>
            </CardContent>
          </Card>

          <Card className="bg-gray-850 border-gray-800">
            <CardContent>
              <h3 className="text-lg font-medium text-white">Configuration Reference</h3>
              <p className="text-gray-300 mt-2">Two options: a JSON file or environment variables. Use the config that fits your deployment pipeline. Example config and .env snippets are provided below for easy copy/paste.</p>
              <div className="mt-4">
                <ConfigSnippet apiKey="YOUR_API_KEY_GOES_HERE" />
              </div>
            </CardContent>
          </Card>
        </section>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold text-white mb-3">Security Guidance</h2>
          <Card className="bg-gray-850 border-gray-800">
            <CardContent>
              <ul className="list-disc pl-5 space-y-2 text-gray-300">
                <li>Never store node RPC credentials in third-party services. Keep them on your server.</li>
                <li>Restrict access to RPC ports with a firewall and bind RPC only to localhost or an internal interface.</li>
                <li>Rotate API keys periodically. Use the /api/renew endpoint for managed renewals.</li>
                <li>Enable mlock/securebuffer features in the Rust-backed library to avoid leaking secrets to swap.</li>
              </ul>
            </CardContent>
          </Card>
        </section>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold text-white mb-3">🔒 Advanced Security Features</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            
            <Card className="bg-gray-850 border-gray-800">
              <CardContent>
                <div className="flex items-center space-x-2 mb-3">
                  <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                  <h3 className="text-lg font-medium text-white">SecureBuffer Protection</h3>
                </div>
                <p className="text-gray-300 text-sm mb-3">
                  Enterprise-grade memory protection for your most sensitive data.
                </p>
                <ul className="list-disc pl-5 space-y-1 text-gray-300 text-sm">
                  <li><strong className="text-white">Memory Locking:</strong> Prevents credentials from being swapped to disk</li>
                  <li><strong className="text-white">Secure Zeroization:</strong> Cryptographically erases sensitive data on cleanup</li>
                  <li><strong className="text-white">Thread-Safe:</strong> Atomic operations prevent race conditions in multi-threaded environments</li>
                  <li><strong className="text-white">Anti-Debugging:</strong> Protects against memory dumps and forensic analysis</li>
                </ul>
                <div className="mt-3 p-2 bg-gray-900 rounded text-xs">
                  <span className="text-green-400">✓ Active:</span> <span className="text-gray-300">Protecting RPC passwords, license keys, and peer secrets</span>
                </div>
              </CardContent>
            </Card>

            <Card className="bg-gray-850 border-gray-800">
              <CardContent>
                <div className="flex items-center space-x-2 mb-3">
                  <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
                  <h3 className="text-lg font-medium text-white">SecureChannel Management</h3>
                </div>
                <p className="text-gray-300 text-sm mb-3">
                  Intelligent connection handling with built-in resilience and monitoring.
                </p>
                <ul className="list-disc pl-5 space-y-1 text-gray-300 text-sm">
                  <li><strong className="text-white">Circuit Breaker:</strong> Automatically isolates failing connections to prevent cascade failures</li>
                  <li><strong className="text-white">Graceful Shutdown:</strong> Ensures clean disconnection and data integrity during restarts</li>
                  <li><strong className="text-white">Health Monitoring:</strong> Real-time connection status and performance metrics</li>
                  <li><strong className="text-white">Auto-Recovery:</strong> Intelligent reconnection with exponential backoff</li>
                </ul>
                <div className="mt-3 p-2 bg-gray-900 rounded text-xs">
                  <span className="text-blue-400">✓ Enhanced:</span> <span className="text-gray-300">99.9% uptime with automatic failover</span>
                </div>
              </CardContent>
            </Card>
          </div>

          <Card className="bg-gradient-to-r from-gray-850 to-gray-800 border-gray-700 mt-4">
            <CardContent>
              <h3 className="text-lg font-medium text-white mb-2">🛡️ Why These Security Features Matter</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                <div>
                  <h4 className="text-white font-medium mb-1">For Exchanges</h4>
                  <p className="text-gray-300">High-volume trading requires bulletproof security. SecureBuffer prevents memory-based attacks, while SecureChannel ensures uninterrupted order processing.</p>
                </div>
                <div>
                  <h4 className="text-white font-medium mb-1">For Enterprises</h4>
                  <p className="text-gray-300">Compliance and audit requirements demand provable security. Our thread-safe design meets SOC 2 standards with forensic-resistant credential handling.</p>
                </div>
                <div>
                  <h4 className="text-white font-medium mb-1">For Custody Services</h4>
                  <p className="text-gray-300">Client funds depend on uncompromised security. Multi-layered protection prevents both external attacks and insider threats from accessing private keys.</p>
                </div>
              </div>
              
              <div className="mt-4 pt-3 border-t border-gray-700">
                <h4 className="text-white font-medium mb-2">📚 Detailed Documentation</h4>
                <div className="flex flex-wrap gap-3">
                  <a href="/docs/SECUREBUFFER_BENEFITS.md" className="inline-flex items-center px-3 py-1 bg-gray-900 hover:bg-gray-800 rounded text-sm text-gray-300 hover:text-white transition-colors">
                    <span className="w-2 h-2 bg-green-500 rounded-full mr-2"></span>
                    SecureBuffer Technical Guide
                  </a>
                  <a href="/docs/SECURECHANNEL_BENEFITS.md" className="inline-flex items-center px-3 py-1 bg-gray-900 hover:bg-gray-800 rounded text-sm text-gray-300 hover:text-white transition-colors">
                    <span className="w-2 h-2 bg-blue-500 rounded-full mr-2"></span>
                    SecureChannel Implementation Guide
                  </a>
                </div>
              </div>
            </CardContent>
          </Card>
        </section>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold text-white mb-3">Examples</h2>
          <Card className="bg-gray-850 border-gray-800">
            <CardContent>
              <p className="text-gray-300">Run locally (example):</p>
              <pre className="bg-gray-900 text-green-300 p-4 rounded mt-3 font-mono overflow-auto text-sm">./bitcoin-sprint --config config.json</pre>
              <div className="mt-4">
                <h4 className="text-sm text-white font-medium">API examples (cURL)</h4>
                <div className="mt-2 grid grid-cols-1 gap-3">
                  <div className="bg-gray-900 p-3 rounded font-mono text-sm text-green-300 flex items-start justify-between">
                    <code>curl -H "Authorization: Bearer &lt;KEY&gt;" https://your-relay.example.com/api/verify</code>
                    <div className="ml-4"><CopyButton text={'curl -H "Authorization: Bearer <KEY>" https://your-relay.example.com/api/verify'} /></div>
                  </div>

                  <div className="bg-gray-900 p-3 rounded font-mono text-sm text-green-300 flex items-start justify-between">
                    <code>{`curl -X POST -H "Authorization: Bearer <KEY>" https://your-relay.example.com/api/renew -d '{"days":30}'`}</code>
                    <div className="ml-4"><CopyButton text={`curl -X POST -H "Authorization: Bearer <KEY>" https://your-relay.example.com/api/renew -d '{"days":30}'`} /></div>
                  </div>
                </div>
                
                <div className="mt-4">
                  <h5 className="text-xs text-gray-400 font-medium mb-2">Sample responses:</h5>
                  <div className="space-y-2">
                    <div className="bg-gray-900 p-2 rounded text-xs">
                      <span className="text-red-400">401 Expired:</span> <code className="text-gray-300">{"{"}"error": "API key expired", "message": "API key expired on 2025-08-01T00:00:00.000Z"{"}"}</code>
                    </div>
                    <div className="bg-gray-900 p-2 rounded text-xs">
                      <span className="text-yellow-400">429 Rate Limited:</span> <code className="text-gray-300">{"{"}"error": "Rate limit exceeded", "message": "Per-minute quota exceeded"{"}"}</code>
                    </div>
                    <div className="bg-gray-900 p-2 rounded text-xs">
                      <span className="text-green-400">200 Success:</span> <code className="text-gray-300">{"{"}"valid": true, "tier": "PRO", "requests": 42, "requestsToday": 15{"}"}</code>
                    </div>
                  </div>
                </div>
              </div>
              <p className="text-gray-300 mt-3">Or build a systemd service to run the relay in production.</p>
            </CardContent>
          </Card>
        </section>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold text-white mb-3">API Key Lifecycle</h2>
          <Card className="bg-gray-850 border-gray-800">
            <CardContent>
              <p className="text-gray-300">API keys have an expiry and usage counters. When an API key expires the relay and web API return HTTP 401 with a clear message <code className="bg-gray-900 px-1 rounded">API key expired</code>. Keys can be renewed via the <code className="bg-gray-900 px-1 rounded">/api/renew</code> endpoint (Authorization: Bearer &lt;key&gt;).</p>
              <ul className="list-disc pl-5 mt-3 text-gray-300">
                <li>Creation: Shown once at signup. Copy immediately.</li>
                <li>Usage: Counters track total requests and daily requests for quota enforcement.</li>
                <li>Expiry: Expired keys are rejected with 401. Renew to extend expiry.</li>
              </ul>
            </CardContent>
          </Card>
        </section>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold text-white mb-3">Troubleshooting</h2>
          <Card className="bg-gray-850 border-gray-800">
            <CardContent>
              <p className="text-gray-300">Common issues:</p>
              <ol className="list-decimal pl-5 mt-3 text-gray-300 space-y-2">
                <li>Key rejected: Check that you copied the full key and that it hasn't expired.</li>
                <li>High latency: Ensure your relay can reach your Bitcoin node with low RTT.</li>
                <li>Rate limited: Upgrade your tier or contact support for higher throughput.</li>
                <li>Edge runtime warnings during development: These are dev-time warnings when middleware uses Node built-ins; safe to ignore in most deployments. Move Node-specific logic out of middleware to resolve permanently.</li>
              </ol>
              
              <div className="mt-4">
                <h4 className="text-sm text-white font-medium mb-2">🔧 Production Setup</h4>
                <div className="bg-gray-900 p-3 rounded text-sm">
                  <p className="text-gray-300 mb-2">For production rate limiting, set up Redis and schedule daily resets:</p>
                  <div className="space-y-1 font-mono text-xs">
                    <div><span className="text-blue-400">REDIS_URL</span>=redis://:password@localhost:6379/0</div>
                    <div><span className="text-green-400">crontab</span>: 5 0 * * * cd /path/to/web && npm run reset:daily</div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </section>

        <footer className="mt-12 text-sm text-gray-400">If you need enterprise onboarding or SLA details, contact sales.</footer>
      </main>
    </div>
  );
}
