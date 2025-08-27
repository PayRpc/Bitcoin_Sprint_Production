# 🚀 Quick RPC Auth Setup (Windows)

## Problem: Special Characters in Passwords
If your `rpcpassword` contains `#`, `"`, or other special characters, Bitcoin Core might fail to parse bitcoin.conf properly.

## ✅ Solution: Use RPC Auth (Recommended)

### Step 1: Generate RPC Auth
Run the batch file to generate secure credentials:

```cmd
# Use default credentials (sprint / MyStrongPassw0rd123!)
gen-rpcauth.bat

# Or specify your own
gen-rpcauth.bat myuser MyCustomPassword123
```

### Step 2: Copy Output to bitcoin.conf
The script will output something like:
```
rpcauth=sprint:a1b2c3d4e5f6g7h8$1234567890abcdef1234567890abcdef12345678901234567890abcdef123456
```

Replace the example line in your `bitcoin.conf` with the real generated line.

### Step 3: Update .env 
Keep the plain password in your `.env` file:
```
BTC_RPC_USER=sprint
BTC_RPC_PASS=MyStrongPassw0rd123!
```

### Step 4: Restart Bitcoin Core
```cmd
# Stop Bitcoin Core
bitcoin-cli.exe stop

# Start with new config
bitcoind.exe -conf=bitcoin.conf
```

## 🔧 Troubleshooting

**Error**: `parse error on line X`
- Check for `#` symbols in passwords
- Make sure rpcauth line is on one line
- No quotes around the rpcauth value

**Error**: `Authorization failed`
- Username/password in .env must match what you used to generate rpcauth
- Case sensitive!

## 🎯 Test Your Setup
```cmd
# Build Sprint API
$env:CGO_ENABLED="0"; go build -o bitcoin-sprint-api.exe ./cmd/sprintd

# Test connection
./bitcoin-sprint-api.exe
```

The API should start without RPC connection errors.
