# 🎲 Smart Contract Deployment Guide
## Entropy Randomness as a Service (ERaaS)

### Quick Deploy Commands

#### Ethereum Deployment
```bash
# Install dependencies
npm install @openzeppelin/contracts hardhat

# Deploy to mainnet
npx hardhat run scripts/deploy-entropy.js --network mainnet

# Deploy to testnet (for testing)
npx hardhat run scripts/deploy-entropy.js --network goerli
```

#### Solana Deployment  
```bash
# Build program
anchor build

# Deploy to mainnet
anchor deploy --provider.cluster mainnet

# Deploy to devnet (for testing)
anchor deploy --provider.cluster devnet
```

### Contract Addresses (After Deployment)
- **Ethereum Mainnet**: `0x...` (TBD)
- **Ethereum Goerli**: `0x...` (TBD)
- **Solana Mainnet**: `...` (TBD)
- **Solana Devnet**: `...` (TBD)

### Revenue Projection
- **Target**: 1M requests/month by Q2 2026
- **Revenue**: $50K-200K monthly recurring revenue
- **Market**: Gaming, DeFi, NFT projects
