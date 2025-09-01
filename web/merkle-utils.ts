// Minimal Merkle (Solidity-compatible, sorted pairs, duplicate last)
// Ethers v5-style utils import; for v6 swap to top-level fns.
import { utils as u } from 'ethers';

const { keccak256, arrayify, solidityPack, getAddress } = u;

// --- Hash helpers ---
const toBytes = (hex: string) => arrayify(hex);
const hashPair = (a: string, b: string, sortPairs = true): string => {
	let A = a.toLowerCase();
	let B = b.toLowerCase();
	if (sortPairs && A > B) [A, B] = [B, A];
	// keccak256(concat(left || right))
	return keccak256(new Uint8Array([...toBytes(A), ...toBytes(B)]));
};

// Leaf = keccak256(abi.encodePacked(address,uint256))
export function hashLeaf(address: string, value: string | number | bigint): string {
	const addr = getAddress(address); // checksum/normalize
	const packed = solidityPack(['address', 'uint256'], [addr, value]);
	return keccak256(packed); // 0x-prefixed hex
}

// Compute Merkle root (sorted pairs, duplicate last for odd layer)
export function merkleRoot(leaves: string[], sortPairs = true): string {
	if (leaves.length === 0) return '0x' + '0'.repeat(64);
	let layer = leaves.slice();
	while (layer.length > 1) {
		const next: string[] = [];
		for (let i = 0; i < layer.length; i += 2) {
			const L = layer[i];
			const R = i + 1 < layer.length ? layer[i + 1] : layer[i]; // duplicate last
			next.push(hashPair(L, R, sortPairs));
		}
		layer = next;
	}
	return layer[0];
}

// Build Merkle proof for a target leaf (sorted pairs, duplicate last)
export function merkleProof(leaves: string[], targetLeaf: string, sortPairs = true): string[] {
	let layer = leaves.slice();
	let idx = layer.findIndex((x) => x.toLowerCase() === targetLeaf.toLowerCase());
	if (idx === -1) throw new Error('Target leaf not found');

	const proof: string[] = [];
	while (layer.length > 1) {
		const sibIdx = idx ^ 1;
		const sibling = sibIdx < layer.length ? layer[sibIdx] : layer[idx];
		proof.push(sibling);

		const next: string[] = [];
		for (let i = 0; i < layer.length; i += 2) {
			const L = layer[i];
			const R = i + 1 < layer.length ? layer[i + 1] : layer[i];
			next.push(hashPair(L, R, sortPairs));
		}
		idx = Math.floor(idx / 2);
		layer = next;
	}
	return proof;
}

// Verify proof without building a tree
export function verifyProof(root: string, leaf: string, proof: string[], sortPairs = true): boolean {
	let h = leaf;
	for (const sib of proof) h = hashPair(h, sib, sortPairs);
	return h.toLowerCase() === root.toLowerCase();
}

// Convenience: from [address,value][] to root + proof for a target
export function generateMerkleProof(
	entries: [string, string | number | bigint][],
	targetAddress: string,
	targetValue: string | number | bigint,
	sortPairs = true
) {
	const leaves = entries.map(([addr, val]) => hashLeaf(addr, val));
	const leaf = hashLeaf(targetAddress, targetValue);
	const proof = merkleProof(leaves, leaf, sortPairs);
	const root = merkleRoot(leaves, sortPairs);
	return { merkleRoot: root, leaf, proof };
}
