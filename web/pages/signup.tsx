import Head from "next/head";
import { useState } from "react";

export default function Signup() {
  const [email, setEmail] = useState("");
  const [company, setCompany] = useState("");
  const [tier, setTier] = useState("starter");
  const [key, setKey] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function createKey(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    setKey(null);
    try {
      const res = await fetch('/api/generate-key', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, company, tier })
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setKey(data.key);
    } catch (err: any) {
      setError(err.message || 'Failed to generate key');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 py-12 flex items-start">
      <Head>
        <title>Get API Key — Bitcoin Sprint</title>
        <meta name="description" content="Generate an API key for Bitcoin Sprint. RPC credentials are not shared with us." />
      </Head>

      <main className="max-w-2xl mx-auto w-full bg-white shadow rounded p-8">
        <h1 className="text-2xl font-bold mb-4">Request an API key</h1>
        <p className="text-sm text-gray-600 mb-6">We will never ask for your Bitcoin Core RPC credentials. Provide contact and company info to generate a license key for testing or production.</p>

        <form onSubmit={createKey} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Email</label>
            <input title="Email" placeholder="you@example.com" type="email" required value={email} onChange={e => setEmail(e.target.value)} className="mt-1 block w-full border rounded p-2" />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">Company</label>
            <input title="Company" placeholder="Your company name (optional)" type="text" value={company} onChange={e => setCompany(e.target.value)} className="mt-1 block w-full border rounded p-2" />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">Tier</label>
            <select title="Tier" value={tier} onChange={e => setTier(e.target.value)} className="mt-1 block w-full border rounded p-2">
              <option value="starter">Starter</option>
              <option value="enterprise">Enterprise</option>
            </select>
          </div>

          <div className="pt-4">
            <button type="submit" disabled={loading} className="bg-blue-600 text-white px-4 py-2 rounded">
              {loading ? 'Generating...' : 'Generate API Key'}
            </button>
          </div>

          {error && <div className="text-red-600">{error}</div>}

          {key && (
            <div className="mt-4 p-4 bg-gray-100 rounded">
              <h3 className="font-medium">Your API Key</h3>
              <p className="break-all font-mono mt-2">{key}</p>
              <p className="text-sm text-gray-600 mt-2">Store this key in your `config.json` as the `license_key` value on your server.</p>
            </div>
          )}
        </form>
      </main>
    </div>
  );
}
