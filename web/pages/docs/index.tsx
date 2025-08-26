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
            <div className="rounded-full overflow-hidden w-[96px] h-[96px]">
              <Image src="/20250823_1017_Bitcoin Sprint Logo_.png" alt="Logo" width={96} height={96} className="object-cover w-full h-full" />
            </div>
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
              <h2 className="text-2xl font-semibold text-white mb-4">🚀 What is Bitcoin Sprint?</h2>
              <p className="text-gray-300 mb-4">
                Bitcoin Sprint is an <strong className="text-white">enterprise-grade Bitcoin block detection and monitoring system</strong> that transforms how businesses interact with the Bitcoin blockchain. We provide real-time Bitcoin block monitoring, secure API access, and enterprise-level integrations that scale from individual developers to large financial institutions.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                <div className="bg-gray-900 p-4 rounded-lg">
                  <h4 className="text-green-400 font-semibold mb-2">✅ With Bitcoin Sprint</h4>
                  <ul className="text-sm text-gray-300 space-y-1">
                    <li>• Plug-and-play integration</li>
                    <li>• Secure memory handling (Rust-powered)</li>
                    <li>• Real-time block notifications</li>
                    <li>• Enterprise support & SLA</li>
                    <li>• Scalable architecture</li>
                  </ul>
                </div>
                <div className="bg-gray-900 p-4 rounded-lg">
                  <h4 className="text-red-400 font-semibold mb-2">❌ Before Bitcoin Sprint</h4>
                  <ul className="text-sm text-gray-300 space-y-1">
                    <li>• Complex Bitcoin Core setup</li>
                    <li>• Manual RPC security management</li>
                    <li>• Unreliable block detection delays</li>
                    <li>• Memory vulnerabilities</li>
                    <li>• No enterprise support</li>
                  </ul>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-gray-850 border-gray-800">
            <CardContent>
              <h2 className="text-2xl font-semibold text-white mb-4">🏢 Service Tiers</h2>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-gray-900 p-4 rounded-lg border border-gray-700">
                  <h3 className="text-lg font-semibold text-white mb-2">🆓 FREE</h3>
                  <p className="text-gray-400 text-sm mb-3">Perfect for developers</p>
                  <ul className="text-sm text-gray-300 space-y-1">
                    <li>• Basic API access</li>
                    <li>• Rate-limited endpoints</li>
                    <li>• Community support</li>
                    <li>• Documentation & examples</li>
                  </ul>
                  <div className="mt-3 pt-3 border-t border-gray-700">
                    <span className="text-green-400 font-semibold">Free forever</span>
                  </div>
                </div>
                
                <div className="bg-gray-900 p-4 rounded-lg border border-blue-500 relative">
                  <div className="absolute -top-2 left-4 bg-blue-500 text-white text-xs px-2 py-1 rounded">
                    POPULAR
                  </div>
                  <h3 className="text-lg font-semibold text-white mb-2">💼 PRO</h3>
                  <p className="text-gray-400 text-sm mb-3">Growing businesses</p>
                  <ul className="text-sm text-gray-300 space-y-1">
                    <li>• 5x higher rate limits</li>
                    <li>• Priority authentication</li>
                    <li>• Enhanced monitoring</li>
                    <li>• Email support (48hr)</li>
                  </ul>
                  <div className="mt-3 pt-3 border-t border-gray-700">
                    <span className="text-blue-400 font-semibold">Monthly subscription</span>
                  </div>
                </div>
                
                <div className="bg-gray-900 p-4 rounded-lg border border-yellow-500">
                  <h3 className="text-lg font-semibold text-white mb-2">🏆 ENTERPRISE</h3>
                  <p className="text-gray-400 text-sm mb-3">Mission-critical ops</p>
                  <ul className="text-sm text-gray-300 space-y-1">
                    <li>• Unlimited requests</li>
                    <li>• 99.9% uptime SLA</li>
                    <li>• 24/7 support</li>
                    <li>• Custom integrations</li>
                  </ul>
                  <div className="mt-3 pt-3 border-t border-gray-700">
                    <span className="text-yellow-400 font-semibold">Custom pricing</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

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
          <h2 className="text-2xl font-semibold text-white mb-3">� Technical Capabilities</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
            
            <Card className="bg-gray-850 border-gray-800">
              <CardContent>
                <h3 className="text-lg font-medium text-white mb-3">⚡ Bitcoin Integration</h3>
                <ul className="list-disc pl-5 space-y-1 text-gray-300 text-sm">
                  <li><strong className="text-white">Real-time block monitoring</strong> with WebSocket and HTTP APIs</li>
                  <li><strong className="text-white">Bitcoin Core RPC proxy</strong> with enhanced security</li>
                  <li><strong className="text-white">P2P network integration</strong> for direct blockchain access</li>
                  <li><strong className="text-white">Transaction monitoring</strong> and analysis tools</li>
                </ul>
              </CardContent>
            </Card>

            <Card className="bg-gray-850 border-gray-800">
              <CardContent>
                <h3 className="text-lg font-medium text-white mb-3">👨‍💻 Developer Experience</h3>
                <ul className="list-disc pl-5 space-y-1 text-gray-300 text-sm">
                  <li><strong className="text-white">RESTful APIs</strong> with comprehensive OpenAPI docs</li>
                  <li><strong className="text-white">Multiple languages</strong> supported (Go, Rust, C++, JS)</li>
                  <li><strong className="text-white">Docker containers</strong> and cloud deployment ready</li>
                  <li><strong className="text-white">Extensive examples</strong> and integration guides</li>
                </ul>
              </CardContent>
            </Card>

            <Card className="bg-gray-850 border-gray-800">
              <CardContent>
                <h3 className="text-lg font-medium text-white mb-3">📊 Monitoring & Analytics</h3>
                <ul className="list-disc pl-5 space-y-1 text-gray-300 text-sm">
                  <li><strong className="text-white">Real-time dashboards</strong> with block and transaction metrics</li>
                  <li><strong className="text-white">Performance monitoring</strong> with Prometheus integration</li>
                  <li><strong className="text-white">Health checks</strong> and automated alerting</li>
                  <li><strong className="text-white">Custom analytics</strong> for enterprise customers</li>
                </ul>
              </CardContent>
            </Card>

            <Card className="bg-gray-850 border-gray-800">
              <CardContent>
                <h3 className="text-lg font-medium text-white mb-3">🌐 Deployment Options</h3>
                <ul className="list-disc pl-5 space-y-1 text-gray-300 text-sm">
                  <li><strong className="text-white">Cloud-hosted</strong> fully managed service</li>
                  <li><strong className="text-white">On-premise enterprise</strong> with source code access</li>
                  <li><strong className="text-white">Hybrid solutions</strong> for data sovereignty</li>
                  <li><strong className="text-white">Global infrastructure</strong> with 99.9% uptime</li>
                </ul>
              </CardContent>
            </Card>
          </div>

          <Card className="bg-gradient-to-r from-blue-900/20 to-purple-900/20 border-blue-500/30">
            <CardContent>
              <h3 className="text-xl font-semibold text-white mb-3">🚀 Quick Integration Example</h3>
              <div className="bg-gray-950 p-4 rounded-lg">
                <pre className="text-green-400 text-sm overflow-x-auto">
{`// Real-time block monitoring
const response = await fetch('https://api.bitcoinsprint.com/v1/blocks/latest', {
  headers: { 'Authorization': 'Bearer YOUR_API_KEY' }
});
const block = await response.json();
console.log(\`New block: \${block.hash}\`);

// WebSocket for real-time updates
const ws = new WebSocket('wss://api.bitcoinsprint.com/v1/ws');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === 'new_block') {
    console.log('Block detected:', data.block);
  }
};`}
                </pre>
              </div>
            </CardContent>
          </Card>
        </section>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold text-white mb-3">�🔒 Advanced Security Features</h2>
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
              
              <div className="mt-3">
                <h4 className="text-white font-medium mb-2">🔗 Go Integration Features</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
                  <div className="bg-gray-900 p-2 rounded">
                    <span className="text-orange-400">⚡ Real-time Monitoring:</span>
                    <p className="text-gray-300 mt-1">HTTP endpoints for connection health, metrics, and Prometheus integration</p>
                  </div>
                  <div className="bg-gray-900 p-2 rounded">
                    <span className="text-purple-400">🔄 CGO Integration:</span>
                    <p className="text-gray-300 mt-1">Seamless FFI bindings between Go services and Rust security components</p>
                  </div>
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

        <section className="mb-8">
          <h2 className="text-2xl font-semibold text-white mb-4">📞 Contact & Support</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            
            <Card className="bg-gradient-to-br from-blue-900/30 to-blue-800/20 border-blue-500/30">
              <CardContent>
                <h3 className="text-lg font-semibold text-white mb-3">💼 Sales & Enterprise</h3>
                <div className="space-y-2 text-sm">
                  <div>
                    <strong className="text-blue-400">Email:</strong>
                    <p className="text-gray-300">enterprise@bitcoinsprint.com</p>
                  </div>
                  <div>
                    <strong className="text-blue-400">Phone:</strong>
                    <p className="text-gray-300">+1-800-BITCOIN-SPRINT</p>
                  </div>
                  <div>
                    <strong className="text-blue-400">Schedule Demo:</strong>
                    <a href="https://calendly.com/bitcoinsprint" className="text-blue-300 hover:text-blue-200 block">
                      calendly.com/bitcoinsprint
                    </a>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-br from-green-900/30 to-green-800/20 border-green-500/30">
              <CardContent>
                <h3 className="text-lg font-semibold text-white mb-3">🛠 Technical Support</h3>
                <div className="space-y-2 text-sm">
                  <div>
                    <strong className="text-green-400">Documentation:</strong>
                    <a href="https://docs.bitcoinsprint.com" className="text-green-300 hover:text-green-200 block">
                      docs.bitcoinsprint.com
                    </a>
                  </div>
                  <div>
                    <strong className="text-green-400">Community:</strong>
                    <a href="https://github.com/PayRpc/Bitcoin_Sprint" className="text-green-300 hover:text-green-200 block">
                      GitHub Repository
                    </a>
                  </div>
                  <div>
                    <strong className="text-green-400">Support Portal:</strong>
                    <a href="https://support.bitcoinsprint.com" className="text-green-300 hover:text-green-200 block">
                      support.bitcoinsprint.com
                    </a>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-br from-purple-900/30 to-purple-800/20 border-purple-500/30">
              <CardContent>
                <h3 className="text-lg font-semibold text-white mb-3">👨‍💻 Developer Resources</h3>
                <div className="space-y-2 text-sm">
                  <div>
                    <strong className="text-purple-400">API Reference:</strong>
                    <p className="text-gray-300">Complete OpenAPI 3.0 spec</p>
                  </div>
                  <div>
                    <strong className="text-purple-400">SDKs:</strong>
                    <p className="text-gray-300">Official libraries for major languages</p>
                  </div>
                  <div>
                    <strong className="text-purple-400">Examples:</strong>
                    <p className="text-gray-300">Production-ready code samples</p>
                  </div>
                  <div>
                    <strong className="text-purple-400">Webhooks:</strong>
                    <p className="text-gray-300">Real-time event notifications</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <Card className="bg-gradient-to-r from-orange-900/20 to-red-900/20 border-orange-500/30 mt-6">
            <CardContent>
              <div className="text-center">
                <h3 className="text-xl font-semibold text-white mb-2">🚀 Ready to Get Started?</h3>
                <p className="text-gray-300 mb-4">
                  Start with our free tier today or schedule an enterprise demo to see Bitcoin Sprint in action.
                </p>
                <div className="flex flex-col sm:flex-row gap-3 justify-center">
                  <a 
                    href="/signup" 
                    className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg font-medium transition-colors"
                  >
                    Start Free Trial
                  </a>
                  <a 
                    href="mailto:enterprise@bitcoinsprint.com" 
                    className="bg-gray-700 hover:bg-gray-600 text-white px-6 py-2 rounded-lg font-medium transition-colors"
                  >
                    Enterprise Demo
                  </a>
                </div>
              </div>
            </CardContent>
          </Card>
        </section>

        <footer className="mt-12 text-sm text-gray-400 text-center">
          <div className="border-t border-gray-800 pt-6">
            <p className="mb-2">
              <strong className="text-white">Bitcoin Sprint</strong> — Making Bitcoin integration sprint-fast for everyone.
            </p>
            <p>
              If you need enterprise onboarding, custom integrations, or SLA details, contact our sales team.
            </p>
          </div>
        </footer>
      </main>
    </div>
  );
}
