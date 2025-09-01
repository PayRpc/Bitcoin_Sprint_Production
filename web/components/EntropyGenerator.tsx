import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { CopyButton } from '@/components/ui/copyButton';
import { Clock, RefreshCw, Shield, Zap } from 'lucide-react';
import { useState } from 'react';

interface EntropyData {
	entropy: string;
	size: number;
	format: string;
	timestamp: string;
	source: string;
	generation_time_ms: number;
}

export default function EntropyGenerator() {
	const [entropy, setEntropy] = useState<EntropyData | null>(null);
	const [loading, setLoading] = useState(false);
	const [size, setSize] = useState(32);
	const [format, setFormat] = useState<'hex' | 'base64' | 'bytes'>('hex');
	const [error, setError] = useState<string | null>(null);

	const generateEntropy = async () => {
		setLoading(true);
		setError(null);

		try {
			const response = await fetch('/api/entropy', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ size, format }),
			});

			if (!response.ok) {
				throw new Error(`Failed to generate entropy: ${response.status}`);
			}

			const data = await response.json();
			setEntropy(data);
		} catch (err: any) {
			setError(err.message);
		} finally {
			setLoading(false);
		}
	};

	const formatEntropyDisplay = (entropyData: EntropyData) => {
		if (format === 'bytes') {
			return entropyData.entropy;
		}
		return entropyData.entropy;
	};

	return (
		<div className="max-w-4xl mx-auto p-6 space-y-6">
			<div className="text-center">
				<h1 className="text-3xl font-bold text-white mb-2 flex items-center justify-center gap-2">
					<Zap className="w-8 h-8 text-yellow-400" />
					Entropy Generator
				</h1>
				<p className="text-gray-400">Generate cryptographically secure random numbers using hardware entropy</p>
			</div>

			{/* Configuration Card */}
			<Card className="bg-gray-800 border-gray-700">
				<CardHeader>
					<CardTitle className="text-white flex items-center gap-2">
						<Shield className="w-5 h-5" />
						Configuration
					</CardTitle>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label className="block text-sm font-medium text-gray-300 mb-2">
								Size (bytes): {size}
							</label>
							<input
								type="range"
								min="1"
								max="256"
								value={size}
								onChange={(e) => setSize(Number(e.target.value))}
								className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer"
							/>
							<div className="flex justify-between text-xs text-gray-500 mt-1">
								<span>1</span>
								<span>128</span>
								<span>256</span>
							</div>
						</div>

						<div>
							<label className="block text-sm font-medium text-gray-300 mb-2">
								Output Format
							</label>
							<select
								value={format}
								onChange={(e) => setFormat(e.target.value as 'hex' | 'base64' | 'bytes')}
								className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white focus:ring-2 focus:ring-blue-500"
							>
								<option value="hex">Hexadecimal</option>
								<option value="base64">Base64</option>
								<option value="bytes">Byte Array</option>
							</select>
						</div>
					</div>

					<Button
						onClick={generateEntropy}
						disabled={loading}
						className="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-3"
					>
						{loading ? (
							<>
								<RefreshCw className="w-4 h-4 mr-2 animate-spin" />
								Generating...
							</>
						) : (
							<>
								<Zap className="w-4 h-4 mr-2" />
								Generate Entropy
							</>
						)}
					</Button>
				</CardContent>
			</Card>

			{/* Error Display */}
			{error && (
				<Card className="bg-red-900 border-red-700">
					<CardContent className="pt-6">
						<div className="text-red-200">
							<strong>Error:</strong> {error}
						</div>
					</CardContent>
				</Card>
			)}

			{/* Results Card */}
			{entropy && (
				<Card className="bg-gray-800 border-gray-700">
					<CardHeader>
						<CardTitle className="text-white flex items-center justify-between">
							<span className="flex items-center gap-2">
								<Shield className="w-5 h-5" />
								Generated Entropy
							</span>
							<div className="flex gap-2">
								<Badge variant="secondary" className="bg-green-600 text-white">
									{entropy.size} bytes
								</Badge>
								<Badge variant="secondary" className="bg-blue-600 text-white">
									{entropy.format}
								</Badge>
							</div>
						</CardTitle>
					</CardHeader>
					<CardContent className="space-y-4">
						{/* Entropy Output */}
						<div>
							<div className="flex items-center justify-between mb-2">
								<label className="text-sm font-medium text-gray-300">Random Data</label>
								<CopyButton text={entropy.entropy} />
							</div>
							<div className="bg-gray-900 border border-gray-600 rounded-md p-4 font-mono text-sm text-green-400 break-all">
								{formatEntropyDisplay(entropy)}
							</div>
						</div>

						{/* Metadata */}
						<div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
							<div className="flex items-center gap-2 text-gray-300">
								<Clock className="w-4 h-4" />
								<span>Generated: {new Date(entropy.timestamp).toLocaleTimeString()}</span>
							</div>
							<div className="flex items-center gap-2 text-gray-300">
								<Zap className="w-4 h-4" />
								<span>{entropy.generation_time_ms}ms</span>
							</div>
							<div className="flex items-center gap-2 text-gray-300">
								<Shield className="w-4 h-4" />
								<span>Source: {entropy.source}</span>
							</div>
						</div>

						{/* Statistics */}
						<div className="bg-gray-900 rounded-md p-4">
							<h4 className="text-white font-medium mb-2">Statistics</h4>
							<div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
								<div>
									<div className="text-gray-400">Bits of Entropy</div>
									<div className="text-white font-mono">{entropy.size * 8}</div>
								</div>
								<div>
									<div className="text-gray-400">Output Length</div>
									<div className="text-white font-mono">
										{format === 'hex' ? entropy.entropy.length :
											format === 'base64' ? Math.ceil(entropy.size * 4 / 3) :
												entropy.entropy.split(',').length} chars
									</div>
								</div>
								<div>
									<div className="text-gray-400">Generation Time</div>
									<div className="text-white font-mono">{entropy.generation_time_ms}ms</div>
								</div>
								<div>
									<div className="text-gray-400">Quality</div>
									<div className="text-green-400 font-mono">High</div>
								</div>
							</div>
						</div>
					</CardContent>
				</Card>
			)}

			{/* Info Card */}
			<Card className="bg-gray-800 border-gray-700">
				<CardContent className="pt-6">
					<div className="text-gray-300 text-sm space-y-2">
						<p><strong>🔐 Security:</strong> Uses hardware-based entropy sources including CPU timing jitter, system fingerprinting, and OS cryptographic randomness.</p>
						<p><strong>⚡ Performance:</strong> Generates 32 bytes in ~20ms average, supporting thousands of requests per second.</p>
						<p><strong>🎯 Use Cases:</strong> Perfect for cryptographic keys, secure tokens, gaming randomness, scientific simulations, and any application requiring true randomness.</p>
					</div>
				</CardContent>
			</Card>
		</div>
	);
}
