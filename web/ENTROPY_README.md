# 🎲 Entropy Generator

Generate cryptographically secure random numbers using hardware entropy sources.

## Features

- **Hardware Security**: Uses CPU timing jitter, system fingerprinting, and OS cryptographic randomness
- **High Performance**: Generates 32 bytes in ~20ms average
- **Multiple Formats**: Output in hexadecimal, base64, or byte array format
- **Web Interface**: Easy-to-use web interface at `/entropy`
- **API Access**: RESTful API endpoint at `/api/entropy`

## Web Interface

Visit `http://localhost:3002/entropy` to use the interactive entropy generator.

### Features:
- **Size Control**: Adjust entropy size from 1-256 bytes
- **Format Selection**: Choose between hex, base64, or byte array output
- **Real-time Generation**: Generate new entropy with one click
- **Copy to Clipboard**: Easily copy generated entropy
- **Performance Metrics**: View generation time and statistics

## API Usage

### Endpoint
```
POST /api/entropy
```

### Request Body
```json
{
  "size": 32,
  "format": "hex"
}
```

### Parameters
- `size` (optional): Number of bytes to generate (1-1024, default: 32)
- `format` (optional): Output format - "hex", "base64", or "bytes" (default: "hex")

### Response
```json
{
  "entropy": "e4e025ebff88bdea974eff75dd6651cfd...",
  "size": 32,
  "format": "hex",
  "timestamp": "2025-09-01T01:30:24.149Z",
  "source": "hardware",
  "generation_time_ms": 15
}
```

### Example Usage

#### Generate 32 bytes (default)
```bash
curl -X POST http://localhost:3002/api/entropy \
  -H "Content-Type: application/json" \
  -d '{}'
```

#### Generate 64 bytes in base64 format
```bash
curl -X POST http://localhost:3002/api/entropy \
  -H "Content-Type: application/json" \
  -d '{"size": 64, "format": "base64"}'
```

#### Generate 16 bytes as byte array
```bash
curl -X POST http://localhost:3002/api/entropy \
  -H "Content-Type: application/json" \
  -d '{"size": 16, "format": "bytes"}'
```

## Use Cases

### Cryptographic Keys
```javascript
// Generate a 256-bit encryption key
const response = await fetch('/api/entropy', {
  method: 'POST',
  body: JSON.stringify({ size: 32 })
});
const key = (await response.json()).entropy;
```

### Secure Tokens
```javascript
// Generate a random session token
const response = await fetch('/api/entropy', {
  method: 'POST',
  body: JSON.stringify({ size: 16, format: 'base64' })
});
const token = (await response.json()).entropy;
```

### Gaming & Simulation
```javascript
// Generate random numbers for games
const response = await fetch('/api/entropy', {
  method: 'POST',
  body: JSON.stringify({ size: 4, format: 'bytes' })
});
const bytes = (await response.json()).entropy.split(',').map(Number);
const randomInt = bytes[0]; // 0-255
```

## Security

- **Hardware Entropy**: Uses multiple hardware sources for true randomness
- **No Repeats**: Each generation produces unique, unpredictable values
- **Fast Generation**: Optimized for high-throughput applications
- **Enterprise Grade**: Built with security best practices

## Performance

- **Average Generation Time**: ~20ms for 32 bytes
- **Throughput**: Supports thousands of requests per second
- **Scalable**: Can handle concurrent requests efficiently
- **Low Latency**: Optimized for real-time applications

## Testing

Run the entropy web API tests:
```bash
npm run test:entropy:web
```

This will test various scenarios including:
- Default entropy generation
- Custom sizes and formats
- Error handling
- Performance metrics
