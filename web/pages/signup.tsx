import Head from "next/head";
import Image from "next/image";
import { useState } from "react";
import ConfigSnippet from "../components/ConfigSnippet";
import { Error as ErrorComponent } from "../components/ui/error";

interface ApiKeyResponse {
  id: string;
  key: string;
  tier: string;
  expiresAt: string;
}

interface ErrorResponse {
  error: string;
  message?: string;
}

export default function Signup() {
  const [email, setEmail] = useState("");
  const [company, setCompany] = useState("");
  const [tier, setTier] = useState("FREE");
  const [response, setResponse] = useState<ApiKeyResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  async function createKey(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    setResponse(null);
    setCopied(false);

    try {
      const res = await fetch('/api/simple-signup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, company: company || undefined, tier })
      });

      if (!res.ok) {
        const errorData: ErrorResponse = await res.json();
        throw new Error(errorData.message || errorData.error || `HTTP ${res.status}`);
      }

      const data: ApiKeyResponse = await res.json();
      setResponse(data);
    } catch (err: any) {
      setError(err.message || 'Failed to generate API key');
    } finally {
      setLoading(false);
    }
  }

  async function copyToClipboard() {
    if (!response?.key) return;
    try {
      await navigator.clipboard.writeText(response.key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      // Fallback for older browsers
      const textArea = document.createElement('textarea');
      textArea.value = response.key;
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand('copy');
      document.body.removeChild(textArea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-950 to-brand-dark py-12 flex items-start relative overflow-hidden">
      <Head>
        <title>Get API Key — Bitcoin Sprint</title>
        <meta name="description" content="Generate a secure API key for Bitcoin Sprint. Keys are persisted, revocable, and include expiration dates." />
      </Head>

      {/* Background Brand Elements */}
      <div className="absolute inset-0 opacity-10">
        <div className="absolute top-10 left-10 w-32 h-32 object-contain">
          <Image
            src="/20250824_0150_Brand Design Elements_simple_compose_01k3d9zhe7e758ny5xyqpm1gmg.png"
            alt=""
            width={128}
            height={128}
            className="object-contain"
          />
        </div>
        <div className="absolute bottom-10 right-10 w-40 h-40 object-contain">
          <Image
            src="/20250824_0150_Brand Design Elements_simple_compose_01k3d9zhe8e4aat8d9480c6tg3.png"
            alt=""
            width={160}
            height={160}
            className="object-contain"
          />
        </div>
      </div>

  <main className="max-w-5xl mx-auto w-full glass shadow-xl rounded-2xl p-8 relative z-10">
        <div className="text-center mb-8">
          <div className="flex justify-center mb-6">
            <div className="rounded-full overflow-hidden w-[220px] h-[220px] drop-shadow-sm">
              <Image
                src="/20250823_1017_Bitcoin Sprint Logo_.png"
                alt="Bitcoin Sprint"
                width={220}
                height={220}
                className="object-cover w-full h-full"
              />
            </div>
          </div>
          <h1 className="text-3xl font-bold text-gradient mb-2">Request an API Key</h1>
          <p className="text-gray-300 leading-relaxed">
            We will never ask for your Bitcoin Core RPC credentials. 
            Provide contact information to generate a secure, expirable license key for testing or production.
          </p>
        </div>

        <form onSubmit={createKey} className="space-y-6">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-200 mb-1">
              Email Address <span className="text-red-400">*</span>
            </label>
            <input
              id="email"
              name="email"
              type="email"
              required
              autoComplete="email"
              placeholder="you@example.com"
              value={email}
              onChange={e => setEmail(e.target.value)}
              className="w-full px-3 py-2 bg-gray-800/50 border border-gray-600 rounded-md shadow-sm text-gray-100 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-brand-gold focus:border-brand-gold"
            />
          </div>

          <div>
            <label htmlFor="company" className="block text-sm font-medium text-gray-200 mb-1">
              Company Name
            </label>
            <input
              id="company"
              name="company"
              type="text"
              autoComplete="organization"
              placeholder="Your company name (optional)"
              value={company}
              onChange={e => setCompany(e.target.value)}
              className="w-full px-3 py-2 bg-gray-800/50 border border-gray-600 rounded-md shadow-sm text-gray-100 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-brand-gold focus:border-brand-gold"
            />
          </div>

          <div>
            <label htmlFor="tier" className="block text-sm font-medium text-gray-200 mb-3">
              API Tier <span className="text-red-400">*</span>
            </label>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* FREE Tier */}
              <label className={`relative cursor-pointer rounded-lg border p-4 transition-colors ${tier === 'FREE' ? 'border-brand-gold bg-brand-gold/10' : 'border-gray-600 bg-gray-800/30'} hover:border-brand-gold/60`}>
                <input
                  type="radio"
                  name="tier"
                  value="FREE"
                  checked={tier === 'FREE'}
                  onChange={e => setTier(e.target.value)}
                  className="sr-only"
                />
                <div className="flex items-center space-x-3">
                  <Image src="/icon-wallet.svg" alt="Wallet" width={32} height={32} />
                  <div className="flex-1">
                    <div className="text-sm font-medium text-gray-100">🟢 Free Tier</div>
                    <div className="text-xs text-gray-400">100 req/min, 100 blocks/day</div>
                  </div>
                </div>
              </label>

              {/* PRO Tier */}
              <label className={`relative cursor-pointer rounded-lg border p-4 transition-colors ${tier === 'PRO' ? 'border-brand-gold bg-brand-gold/10' : 'border-gray-600 bg-gray-800/30'} hover:border-brand-gold/60`}>
                <input
                  type="radio"
                  name="tier"
                  value="PRO"
                  checked={tier === 'PRO'}
                  onChange={e => setTier(e.target.value)}
                  className="sr-only"
                />
                <div className="flex items-center space-x-3">
                  <Image src="/icon-exchange.svg" alt="Exchange" width={32} height={32} />
                  <div className="flex-1">
                    <div className="text-sm font-medium text-gray-100">🔵 Pro Tier</div>
                    <div className="text-xs text-gray-400">2,000 req/min, unlimited blocks</div>
                  </div>
                </div>
              </label>

              {/* ENTERPRISE Tier */}
              <label className={`relative cursor-pointer rounded-lg border p-4 transition-colors ${tier === 'ENTERPRISE' ? 'border-brand-gold bg-brand-gold/10' : 'border-gray-600 bg-gray-800/30'} hover:border-brand-gold/60`}>
                <input
                  type="radio"
                  name="tier"
                  value="ENTERPRISE"
                  checked={tier === 'ENTERPRISE'}
                  onChange={e => setTier(e.target.value)}
                  className="sr-only"
                />
                <div className="flex items-center space-x-3">
                  <Image src="/icon-enterprise.svg" alt="Enterprise" width={32} height={32} />
                  <div className="flex-1">
                    <div className="text-sm font-medium text-gray-100">🟣 Enterprise</div>
                    <div className="text-xs text-gray-400">20,000 req/min, full API suite</div>
                  </div>
                </div>
              </label>

              {/* ENTERPRISE_PLUS Tier */}
              <label className={`relative cursor-pointer rounded-lg border p-4 transition-colors ${tier === 'ENTERPRISE_PLUS' ? 'border-brand-gold bg-brand-gold/10' : 'border-gray-600 bg-gray-800/30'} hover:border-brand-gold/60`}>
                <input
                  type="radio"
                  name="tier"
                  value="ENTERPRISE_PLUS"
                  checked={tier === 'ENTERPRISE_PLUS'}
                  onChange={e => setTier(e.target.value)}
                  className="sr-only"
                />
                <div className="flex items-center space-x-3">
                  <Image src="/icon-enterprise.svg" alt="Enterprise Plus" width={32} height={32} />
                  <div className="flex-1">
                    <div className="text-sm font-medium text-gray-100">🔴 Enterprise+</div>
                    <div className="text-xs text-gray-400">100k+ req/min, dedicated infra</div>
                  </div>
                </div>
              </label>
            </div>
          </div>

          <div className="pt-4">
            <button
              type="submit"
              disabled={loading || !email.trim()}
              className="w-full btn disabled:opacity-50 disabled:cursor-not-allowed py-3 px-4 shadow-glow transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-brand-gold focus:ring-offset-2 focus:ring-offset-brand-dark transform hover:scale-[1.02] disabled:hover:scale-100"
            >
              {loading ? (
                <span className="flex items-center justify-center">
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-brand-dark" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  Generating Key...
                </span>
              ) : (
                'Generate API Key'
              )}
            </button>
          </div>

          {error && (
            <ErrorComponent 
              message={error}
              onRetry={() => setError(null)}
            />
          )}

          {response && (
            <div className="p-6 bg-gradient-to-r from-green-900/30 to-emerald-900/30 border border-green-600/50 rounded-lg relative overflow-hidden">
              {/* Subtle brand element in success state */}
              <div className="absolute top-2 right-2 opacity-20">
                <Image src="/icon-wallet.svg" alt="" width={48} height={48} className="h-12 w-12" />
              </div>
              
              <div className="flex items-start relative z-10">
                <div className="flex-shrink-0">
                  <svg className="h-6 w-6 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <div className="ml-3 flex-1">
                  <h3 className="text-lg font-medium text-green-300 mb-3">API Key Generated Successfully!</h3>
                  
                  <div className="bg-gray-800/70 border border-green-600/30 rounded-md p-4 mb-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm font-medium text-gray-200">Your API Key:</span>
                      <button
                        type="button"
                        onClick={copyToClipboard}
                        className="inline-flex items-center px-2 py-1 text-xs font-medium text-brand-gold hover:text-brand-orange focus:outline-none"
                      >
                        {copied ? (
                          <>
                            <svg className="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                              <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                            </svg>
                            Copied!
                          </>
                        ) : (
                          <>
                            <svg className="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                              <path d="M8 2a1 1 0 000 2h2a1 1 0 100-2H8z" />
                              <path d="M3 5a2 2 0 012-2 3 3 0 003 3h6a3 3 0 003-3 2 2 0 012 2v6h-4.586l1.293-1.293a1 1 0 00-1.414-1.414l-3 3a1 1 0 000 1.414l3 3a1 1 0 001.414-1.414L14.586 13H19v3a2 2 0 01-2 2H5a2 2 0 01-2-2V5zM15 11.586V13a1 1 0 11-2 0v-1.586l.293.293a1 1 0 001.414 0z" />
                            </svg>
                            Copy
                          </>
                        )}
                      </button>
                    </div>
                    <code className="block w-full p-2 bg-gray-900/70 border border-gray-600 rounded font-mono text-sm break-all select-all text-brand-gold">
                      {response.key}
                    </code>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4 text-sm">
                    <div>
                      <span className="font-medium text-gray-200">Tier:</span>
                      <span className="ml-2 badge">
                        {response.tier}
                      </span>
                    </div>
                    <div>
                      <span className="font-medium text-gray-200">Expires:</span>
                      <span className="ml-2 text-gray-300">
                        {new Date(response.expiresAt).toLocaleDateString('en-US', {
                          year: 'numeric',
                          month: 'long',
                          day: 'numeric'
                        })}
                      </span>
                    </div>
                  </div>

                  {/* Configuration Options */}
                  <div className="mt-6">
                    <h4 className="text-lg font-medium text-gray-200 mb-4">🔧 Configuration Options</h4>
                    <ConfigSnippet apiKey={response.key} />
                  </div>

                  <div className="mt-6 bg-brand-orange/20 border border-brand-orange/50 rounded-md p-3">
                    <h4 className="text-sm font-medium text-brand-orange mb-2">⚠️ Important Security Notice</h4>
                    <ul className="text-sm text-orange-200 space-y-1">
                      <li>• This key will only be shown once - copy it now</li>
                      <li>• Never commit API keys to version control</li>
                      <li>• Keys are revocable if compromised</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          )}
        </form>
      </main>
    </div>
  );
}
