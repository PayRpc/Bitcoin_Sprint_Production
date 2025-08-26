import { Card, CardContent } from "@/components/ui/card";
import { motion } from "framer-motion";
import { BarChart3, Shield, Zap } from "lucide-react";
import Head from "next/head";
import Image from "next/image";
import Link from "next/link";

export default function ApiFrontPage() {
  const configExample = `{
  "license_key": "YOUR_API_KEY",
  "rpc_nodes": ["http://localhost:8332"],
  "rpc_user": "bitcoinrpc",
  "rpc_pass": "mypassword",
  "turbo_mode": true
}`;

  return (
    <>
      <Head>
        <title>Bitcoin Sprint — Fast, secure Bitcoin API</title>
        <meta name="description" content="Bitcoin Sprint: low-latency Bitcoin relay for sub-second block detection. Keep RPC credentials on your server; use our API key to access the relay." />
        <meta property="og:title" content="Bitcoin Sprint — Fast, secure Bitcoin API" />
        <meta property="og:description" content="Low-latency relay with secure API key isolation. RPC credentials stay on your server." />
  <meta property="og:image" content="/20250823_1017_Bitcoin Sprint Logo_.png" />
      </Head>
    
    <div className="min-h-screen bg-gradient-to-b from-gray-900 to-black text-white">
      {/* Hero Section */}
      <section className="text-center py-20 px-6">
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
          className="flex flex-col items-center"
        >
          <div className="rounded-full overflow-hidden w-[220px] h-[220px]">
            <Image
              src="/20250823_1017_Bitcoin Sprint Logo_.png"
              alt="Bitcoin Sprint Logo"
              width={220}
              height={220}
              priority
              className="object-cover w-full h-full"
            />
          </div>
          <h1 className="text-5xl font-bold mt-6 mb-4">Bitcoin Sprint</h1>
          <p className="text-xl text-gray-300 mb-6">First to the Block, First to Profit.</p>
          <div className="flex justify-center gap-4">
            <Link
              href="/signup"
              className="inline-flex items-center justify-center rounded-2xl px-6 py-3 bg-orange-500 hover:bg-orange-600 text-white font-medium"
              aria-label="Get API Key"
            >
              Get API Key
            </Link>

            <Link
              href="/docs"
              className="inline-flex items-center justify-center rounded-2xl px-6 py-3 border border-white/20 text-white font-medium hover:bg-white/5"
              aria-label="View Docs"
            >
              View Docs
            </Link>
          </div>
        </motion.div>
      </section>

      {/* Who Benefits */}
      <section className="py-16 px-8 grid md:grid-cols-3 gap-8 max-w-6xl mx-auto">
        <Card className="bg-gray-800/50 border-gray-700">
          <CardContent className="p-6 text-center">
            <Image src="/icon-wallet.svg" alt="Wallet Icon" width={50} height={50} className="mx-auto mb-4" />
            <h3 className="text-2xl font-semibold mb-3">Wallet Users & Providers</h3>
            <ul className="space-y-2 text-gray-300">
              <li>Faster deposit confirmations</li>
              <li>Instant mempool updates for pending transactions</li>
              <li>Secrets kept in protected memory on the user's server</li>
              <li>No node maintenance required</li>
            </ul>
          </CardContent>
        </Card>

        <Card className="bg-gray-800/50 border-gray-700">
          <CardContent className="p-6 text-center">
            <Image src="/icon-exchange.svg" alt="Exchange Icon" width={50} height={50} className="mx-auto mb-4" />
            <h3 className="text-2xl font-semibold mb-3">Exchanges & Trading</h3>
            <ul className="space-y-2 text-gray-300">
              <li>Sub-second block detection for faster deposits</li>
              <li>Trade execution ahead of slower peers</li>
              <li>Secure API key isolation with Rust SecureBuffer on our relay</li>
              <li>Enterprise SLAs for uptime & throughput</li>
            </ul>
          </CardContent>
        </Card>

        <Card className="bg-gray-800/50 border-gray-700">
          <CardContent className="p-6 text-center">
            <Image src="/icon-enterprise.svg" alt="Enterprise Icon" width={50} height={50} className="mx-auto mb-4" />
            <h3 className="text-2xl font-semibold mb-3">Enterprises</h3>
            <ul className="space-y-2 text-gray-300">
              <li>Compliance-ready: memory zeroization & mlock</li>
              <li>Cost-effective API replaces node ops</li>
              <li>Scalable licensing tiers for any size</li>
              <li>Built-in metrics & observability</li>
            </ul>
          </CardContent>
        </Card>
      </section>

      {/* Configuration / How users provide credentials */}
      <section className="py-12 px-6 max-w-4xl mx-auto">
        <h2 className="text-3xl font-bold mb-4">Where to put your credentials</h2>
        <p className="text-gray-300 mb-4">
          Users keep their Bitcoin Core / RPC credentials on their own server. Do not collect their RPC credentials in the front-end. Do not send RPC credentials to our backend.
        </p>

        <p className="text-gray-300 mb-4">
          Example `config.json` (placed on the customer's server):
        </p>

        <pre className="bg-gray-900 p-4 rounded text-sm overflow-auto text-left"><code>{configExample}</code></pre>

        <p className="text-gray-300 mt-4">
          Example run command (on the customer's server):
        </p>
        <pre className="bg-gray-900 p-4 rounded text-sm overflow-auto text-left"><code>./bitcoin-sprint --config config.json</code></pre>

        <div className="mt-6 text-gray-400">
          <strong>Notes:</strong>
          <ul className="list-disc list-inside">
            <li>License keys (API keys) are generated by our front-end and used to authenticate to the relay.</li>
            <li>RPC credentials stay on your server and only your server talks to your node.</li>
            <li>The front-end should only guide users and offer the ability to generate an API key; it must not accept node RPC credentials.</li>
            <li>See the Docs for full setup and configuration examples.</li>
          </ul>
        </div>
      </section>

      {/* Benchmarks */}
      <section className="py-20 bg-gray-950 px-6 text-center">
        <h2 className="text-3xl font-bold mb-10">Benchmark Highlights</h2>
        <div className="grid md:grid-cols-3 gap-8 max-w-5xl mx-auto">
          <Card className="bg-gray-800/70 border-gray-700">
            <CardContent className="p-6 text-center">
              <Zap className="mx-auto mb-3 text-yellow-400" size={36} />
              <p className="text-xl font-semibold">⚡ 200 ms Block Detection</p>
              <p className="text-gray-400">vs 10–30 s with Bitcoin Core</p>
            </CardContent>
          </Card>
          <Card className="bg-gray-800/70 border-gray-700">
            <CardContent className="p-6 text-center">
              <BarChart3 className="mx-auto mb-3 text-green-400" size={36} />
              <p className="text-xl font-semibold">📊 200k+ Requests/Second</p>
              <p className="text-gray-400">20× faster than standard JSON-RPC</p>
            </CardContent>
          </Card>
          <Card className="bg-gray-800/70 border-gray-700">
            <CardContent className="p-6 text-center">
              <Shield className="mx-auto mb-3 text-blue-400" size={36} />
              <p className="text-xl font-semibold">🛡️ Rust SecureBuffer</p>
              <p className="text-gray-400">Secrets locked in RAM, zeroized on drop, never paged — even under stress</p>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Footer */}
      <footer className="text-center py-10 text-gray-500">
        <p>Bitcoin Sprint: First to the block, first to profit.</p>
        <p className="mt-2 text-sm">© 2025 Bitcoin Sprint Systems</p>
      </footer>
    </div>
    </>
  );
}
