import Head from "next/head";
import Image from "next/image";

const configExample = `{
  "license_key": "YOUR_API_KEY",
  "rpc_nodes": ["http://localhost:8332"],
  "rpc_user": "bitcoinrpc",
  "rpc_pass": "mypassword",
  "turbo_mode": true
}`;

export default function DocsPage() {
  return (
    <div className="min-h-screen bg-white text-gray-900">
      <Head>
        <title>Bitcoin Sprint - Docs</title>
        <meta name="description" content="Documentation and configuration for Bitcoin Sprint. Keep your RPC credentials on your own server; the relay uses an API key." />
        <meta property="og:title" content="Bitcoin Sprint Docs" />
        <meta property="og:description" content="How to configure Bitcoin Sprint and where to store RPC credentials." />
      </Head>

      <main className="max-w-4xl mx-auto py-16 px-6">
        <header className="text-center mb-12">
          <Image src="/logo-bitcoin-sprint.svg" alt="Logo" width={120} height={120} />
          <h1 className="text-4xl font-bold mt-6">Documentation</h1>
          <p className="text-gray-600 mt-2">Configuration and security guidance for deploying Bitcoin Sprint.</p>
        </header>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold mb-3">Where credentials live</h2>
          <p className="text-gray-700 mb-3">Do not send your node's RPC username/password to our website or API. Keep those credentials on your server and configure `bitcoin-sprint` to talk only to your node.</p>
        </section>

        <section className="mb-8">
          <h2 className="text-2xl font-semibold mb-3">Example config.json</h2>
          <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto"><code>{configExample}</code></pre>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-3">Run locally</h2>
          <p className="text-gray-700 mb-3">Place the example `config.json` on your server and run:</p>
          <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto"><code>./bitcoin-sprint --config config.json</code></pre>
        </section>

        <footer className="mt-12 text-sm text-gray-500">If you need enterprise onboarding or SLA details, contact sales.</footer>
      </main>
    </div>
  );
}
