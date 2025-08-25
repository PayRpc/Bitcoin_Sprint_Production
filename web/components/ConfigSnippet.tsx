import { Check, Copy, Shield } from "lucide-react";
import { useState } from "react";

export default function ConfigSnippet({ apiKey }: { apiKey: string }) {
  const [copied, setCopied] = useState<"config" | "env" | null>(null);

  const configJson = `{
  "license_key": "${apiKey}",
  "rpc_nodes": ["http://localhost:8332"],
  "rpc_user": "your-rpc-user",
  "rpc_pass": "your-rpc-password",
  "turbo_mode": true
}`;

  const envFile = `# Bitcoin Sprint Example .env.local
LICENSE_KEY=${apiKey}
RPC_NODES=http://localhost:8332
RPC_USER=your-rpc-user
RPC_PASS=your-rpc-password
PEER_SECRET=your-shared-peer-secret
TURBO_MODE=true`;

  const copyToClipboard = async (text: string, type: "config" | "env") => {
    await navigator.clipboard.writeText(text);
    setCopied(type);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <div className="space-y-6">
      {/* JSON Config */}
      <div>
        <h3 className="font-semibold mb-2 text-gray-200">Option 1: config.json</h3>
        <div className="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm relative overflow-auto">
          <pre className="whitespace-pre text-sm">{configJson}</pre>
          <button
            onClick={() => copyToClipboard(configJson, "config")}
            className="absolute top-2 right-2 flex items-center space-x-1 text-xs text-gray-300 hover:text-white"
          >
            {copied === "config" ? <Check size={14} /> : <Copy size={14} />}
            <span>{copied === "config" ? "Copied!" : "Copy"}</span>
          </button>
        </div>
      </div>

      {/* ENV Config */}
      <div>
        <h3 className="font-semibold mb-2 text-gray-200">Option 2: .env.local</h3>
        <div className="bg-gray-900 text-blue-400 p-4 rounded-lg font-mono text-sm relative overflow-auto">
          <pre className="whitespace-pre text-sm">{envFile}</pre>
          <button
            onClick={() => copyToClipboard(envFile, "env")}
            className="absolute top-2 right-2 flex items-center space-x-1 text-xs text-gray-300 hover:text-white"
          >
            {copied === "env" ? <Check size={14} /> : <Copy size={14} />}
            <span>{copied === "env" ? "Copied!" : "Copy"}</span>
          </button>
        </div>
      </div>

      {/* Memory Safety Assurance */}
      <div className="flex items-start space-x-2 bg-gray-800 p-3 rounded-lg text-sm text-gray-200">
        <Shield className="text-green-400 mt-0.5" size={16} />
        <p>
          <strong>SecureBuffer Enabled:</strong> RPC credentials and license keys are{" "}
          <span className="text-green-400">locked in memory (mlock)</span> and{" "}
          <span className="text-green-400">zeroized after use</span>. They never leave
          your server or appear in logs.
        </p>
      </div>
    </div>
  );
}
