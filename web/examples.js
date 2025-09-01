/**
 * Bitcoin Sprint Web App - Secure Usage Examples
 * Demonstrates how to use the secure API client in your frontend components
 */

import { BitcoinSprintApiClient } from './lib/api-client.js';

// Initialize the API client with your API key
const apiClient = new BitcoinSprintApiClient();

// Example 1: Basic entropy generation with automatic authentication
async function generateEntropyExample() {
	try {
		// The client automatically handles authentication and rate limiting
		const result = await apiClient.generateEntropy({
			size: 32,
			format: 'hex'
		});

		console.log('Generated entropy:', result.entropy);
		console.log('Tier used:', result.tier);
		console.log('Rate limit remaining:', result.rateLimitRemaining);

		return result;
	} catch (error) {
		console.error('Error generating entropy:', error.message);
		// Handle different error types
		if (error.code === 'RATE_LIMIT_EXCEEDED') {
			console.log('Rate limit exceeded, please wait before retrying');
		} else if (error.code === 'INVALID_API_KEY') {
			console.log('Please check your API key configuration');
		}
	}
}

// Example 2: Testing connection to backend
async function testConnectionExample() {
	try {
		const isConnected = await apiClient.testConnection();
		console.log('Backend connection:', isConnected ? '✅ Connected' : '❌ Disconnected');
		return isConnected;
	} catch (error) {
		console.error('Connection test failed:', error.message);
		return false;
	}
}

// Example 3: React component using the secure API client
function EntropyGenerator({ onEntropyGenerated }) {
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState(null);
	const [entropy, setEntropy] = useState(null);

	const handleGenerate = async () => {
		setLoading(true);
		setError(null);

		try {
			const result = await apiClient.generateEntropy({
				size: 64,
				format: 'base64'
			});

			setEntropy(result.entropy);
			onEntropyGenerated?.(result);
		} catch (err) {
			setError(err.message);
		} finally {
			setLoading(false);
		}
	};

	return (
		<div className="entropy-generator">
			<button
				onClick={handleGenerate}
				disabled={loading}
				className="generate-btn"
			>
				{loading ? 'Generating...' : 'Generate Secure Entropy'}
			</button>

			{error && (
				<div className="error-message">
					Error: {error}
				</div>
			)}

			{entropy && (
				<div className="entropy-result">
					<h3>Generated Entropy:</h3>
					<pre>{entropy}</pre>
				</div>
			)}
		</div>
	);
}

// Example 4: Setting up API key in your application
function setupApiKey() {
	// Option 1: Set API key programmatically
	apiClient.setApiKey('your-api-key-here');

	// Option 2: The client will automatically use the API key from localStorage
	// if it was previously set

	// Option 3: Use environment variables (recommended for production)
	// NEXT_PUBLIC_API_KEY=your-api-key-here
}

// Example 5: Handling different tiers programmatically
async function handleTierBasedRequests() {
	try {
		// Free tier - limited to 32 bytes
		const freeResult = await apiClient.generateEntropy({ size: 32 });

		// Pro tier - up to 128 bytes
		const proResult = await apiClient.generateEntropy({ size: 128 });

		// Enterprise tier - up to 1024 bytes
		const enterpriseResult = await apiClient.generateEntropy({ size: 1024 });

		console.log('All tiers working:', {
			free: freeResult.tier,
			pro: proResult.tier,
			enterprise: enterpriseResult.tier
		});
	} catch (error) {
		console.error('Tier handling error:', error);
	}
}

// Example 6: Error handling and retry logic
async function robustEntropyGeneration(retries = 3) {
	for (let attempt = 1; attempt <= retries; attempt++) {
		try {
			const result = await apiClient.generateEntropy({
				size: 64,
				format: 'hex'
			});

			return result;
		} catch (error) {
			console.log(`Attempt ${attempt} failed:`, error.message);

			if (error.code === 'RATE_LIMIT_EXCEEDED' && attempt < retries) {
				// Wait before retrying (exponential backoff)
				const waitTime = Math.pow(2, attempt) * 1000;
				console.log(`Waiting ${waitTime}ms before retry...`);
				await new Promise(resolve => setTimeout(resolve, waitTime));
				continue;
			}

			throw error;
		}
	}
}

// Example 7: Monitoring API usage
function monitorApiUsage() {
	// The API client automatically tracks rate limits
	// You can access current usage information
	const usage = apiClient.getRateLimitStatus();
	console.log('Current API usage:', usage);

	// Example usage object:
	// {
	//   remaining: 95,
	//   limit: 100,
	//   resetTime: 1640995200000,
	//   tier: 'free'
	// }
}

// Export examples for use in your application
export {
	EntropyGenerator, generateEntropyExample, handleTierBasedRequests, monitorApiUsage, robustEntropyGeneration, setupApiKey, testConnectionExample
};

// Usage in your main application:
//
// import {
//   EntropyGenerator,
//   setupApiKey,
//   testConnectionExample
// } from './examples.js';
//
// // Initialize
// setupApiKey();
//
// // Test connection on app start
// testConnectionExample().then(connected => {
//   if (connected) {
//     console.log('Backend is ready!');
//   }
// });
//
// // Use in React component
// function App() {
//   return (
//     <div>
//       <h1>Bitcoin Sprint Secure Web App</h1>
//       <EntropyGenerator onEntropyGenerated={(result) => {
//         console.log('New entropy generated:', result);
//       }} />
//     </div>
//   );
// }
