// Merkle Tree Utilities for Enterprise Storage Validation
// Solidity-compatible hashing using ethers.js

class MerkleUtils {
	constructor() {
		// Initialize ethers if available
		this.ethers = window.ethers;
	}

	// Hash a leaf node (Solidity-compatible)
	hashLeaf(data) {
		if (this.ethers) {
			return this.ethers.utils.keccak256(this.ethers.utils.toUtf8Bytes(data));
		}
		// Fallback to Web Crypto API
		return crypto.subtle.digest('SHA-256', new TextEncoder().encode(data))
			.then(hash => {
				return '0x' + Array.from(new Uint8Array(hash))
					.map(b => b.toString(16).padStart(2, '0'))
					.join('');
			});
	}

	// Calculate Merkle root from leaves
	async merkleRoot(leaves) {
		if (leaves.length === 0) return null;
		if (leaves.length === 1) return await this.hashLeaf(leaves[0]);

		let layer = leaves;
		while (layer.length > 1) {
			const newLayer = [];
			for (let i = 0; i < layer.length; i += 2) {
				const left = layer[i];
				const right = i + 1 < layer.length ? layer[i + 1] : left;
				const combined = left + right;
				const hash = await this.hashLeaf(combined);
				newLayer.push(hash);
			}
			layer = newLayer;
		}
		return layer[0];
	}

	// Generate Merkle proof for a leaf
	async merkleProof(leaves, targetIndex) {
		if (targetIndex < 0 || targetIndex >= leaves.length) {
			throw new Error('Invalid target index');
		}

		const proof = [];
		let layer = leaves;
		let index = targetIndex;

		while (layer.length > 1) {
			const newLayer = [];
			const isLeft = index % 2 === 0;
			const siblingIndex = isLeft ? index + 1 : index - 1;

			if (siblingIndex < layer.length) {
				proof.push({
					hash: layer[siblingIndex],
					position: isLeft ? 'right' : 'left'
				});
			}

			for (let i = 0; i < layer.length; i += 2) {
				const left = layer[i];
				const right = i + 1 < layer.length ? layer[i + 1] : left;
				const combined = left + right;
				const hash = await this.hashLeaf(combined);
				newLayer.push(hash);
			}

			layer = newLayer;
			index = Math.floor(index / 2);
		}

		return proof;
	}

	// Verify a Merkle proof
	async verifyProof(root, leaf, proof) {
		let computedHash = await this.hashLeaf(leaf);

		for (const { hash, position } of proof) {
			const combined = position === 'left'
				? hash + computedHash
				: computedHash + hash;
			computedHash = await this.hashLeaf(combined);
		}

		return computedHash === root;
	}

	// Generate complete Merkle proof data
	async generateMerkleProof(leaves, targetIndex) {
		const root = await this.merkleRoot(leaves);
		const proof = await this.merkleProof(leaves, targetIndex);

		return {
			root,
			proof,
			leaf: leaves[targetIndex],
			index: targetIndex
		};
	}

	// Split file into chunks for Merkle tree
	async chunkFile(file, chunkSize = 1024 * 1024) { // 1MB chunks
		const chunks = [];
		const totalChunks = Math.ceil(file.size / chunkSize);

		for (let i = 0; i < totalChunks; i++) {
			const start = i * chunkSize;
			const end = Math.min(start + chunkSize, file.size);
			const chunk = file.slice(start, end);

			const buffer = await chunk.arrayBuffer();
			const hash = await crypto.subtle.digest('SHA-256', buffer);
			const hashHex = '0x' + Array.from(new Uint8Array(hash))
				.map(b => b.toString(16).padStart(2, '0'))
				.join('');

			chunks.push(hashHex);
		}

		return chunks;
	}

	// Generate commitment for file chunks
	async generateCommitment(file, chunkSize = 1024 * 1024) {
		const chunks = await this.chunkFile(file, chunkSize);
		const root = await this.merkleRoot(chunks);

		return {
			root,
			chunks,
			totalSize: file.size,
			chunkSize,
			chunkCount: chunks.length
		};
	}
}

// Global instance
window.merkleUtils = new MerkleUtils();
