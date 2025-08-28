// Package securebuf provides enterprise-grade memory protection and Bitcoin-specific optimizations
// This example shows how to integrate the full SecureBuffer API into Bitcoin Sprint

package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
)

// DemonstrateSecureBufferIntegration shows comprehensive usage of SecureBuffer features
func DemonstrateSecureBufferIntegration() {
	fmt.Println("🔐 Bitcoin Sprint SecureBuffer Integration Demo")
	fmt.Println("===============================================")

	// === 1. BASIC ENTROPY OPERATIONS ===
	fmt.Println("\n1. Basic Entropy Operations:")

	// Generate fast entropy
	fastEntropy, err := securebuf.FastEntropy()
	if err != nil {
		log.Printf("Fast entropy error: %v", err)
	} else {
		fmt.Printf("   Fast Entropy (32 bytes): %s\n", hex.EncodeToString(fastEntropy))
	}

	// Get system fingerprint
	fingerprint, err := securebuf.SystemFingerprint()
	if err != nil {
		log.Printf("Fingerprint error: %v", err)
	} else {
		fmt.Printf("   System Fingerprint: %s\n", hex.EncodeToString(fingerprint[:16]))
	}

	// Get CPU temperature for entropy
	temp, err := securebuf.GetCPUTemperature()
	if err != nil {
		log.Printf("CPU temperature error: %v", err)
	} else {
		fmt.Printf("   CPU Temperature: %.2f°C\n", temp)
	}

	// === 2. ENTROPY BUFFERS ===
	fmt.Println("\n2. Entropy-Filled Buffers:")

	// Create buffer with fast entropy
	entropyBuf, err := securebuf.NewWithFastEntropy(64)
	if err != nil {
		log.Printf("Entropy buffer error: %v", err)
		return
	}
	defer entropyBuf.Free()

	// Read some entropy data
	entropyData, err := entropyBuf.ReadToSlice()
	if err != nil {
		log.Printf("Read entropy error: %v", err)
	} else {
		fmt.Printf("   Entropy Buffer (first 16 bytes): %s\n", hex.EncodeToString(entropyData[:16]))
	}

	// === 3. BITCOIN HEADER ENTROPY ===
	fmt.Println("\n3. Bitcoin Header Entropy:")

	// Mock Bitcoin block headers (80 bytes each)
	mockHeaders := [][]byte{
		make([]byte, 80), // Header 1
		make([]byte, 80), // Header 2
	}

	// Fill headers with mock data
	for i, header := range mockHeaders {
		for j := range header {
			header[j] = byte(i*j + 42) // Mock header data
		}
	}

	// Generate hybrid entropy with Bitcoin headers
	hybridEntropy, err := securebuf.HybridEntropy(mockHeaders)
	if err != nil {
		log.Printf("Hybrid entropy error: %v", err)
	} else {
		fmt.Printf("   Hybrid Entropy (with headers): %s\n", hex.EncodeToString(hybridEntropy[:16]))
	}

	// Create buffer with hybrid entropy
	hybridBuf, err := securebuf.NewWithHybridEntropy(128, mockHeaders)
	if err != nil {
		log.Printf("Hybrid buffer error: %v", err)
	} else {
		defer hybridBuf.Free()
		fmt.Printf("   Hybrid Buffer Created: %d bytes capacity\n", hybridBuf.Capacity())
	}

	// === 4. ENTERPRISE SECURITY ===
	fmt.Println("\n4. Enterprise Security Features:")

	// Create enterprise buffer with high security
	enterpriseBuf, err := securebuf.NewWithSecurityLevel(256, securebuf.SecurityEnterprise)
	if err != nil {
		log.Printf("Enterprise buffer error: %v", err)
	} else {
		defer enterpriseBuf.Free()
		fmt.Printf("   Enterprise Buffer Created: %d bytes\n", enterpriseBuf.Capacity())

		// Enable tamper detection
		if err := enterpriseBuf.EnableTamperDetection(); err != nil {
			log.Printf("Tamper detection error: %v", err)
		} else {
			fmt.Printf("   Tamper Detection: ENABLED\n")
		}

		// Check if tampered
		if enterpriseBuf.IsTampered() {
			fmt.Printf("   Tamper Status: TAMPERED!\n")
		} else {
			fmt.Printf("   Tamper Status: CLEAN\n")
		}

		// Try to bind to hardware (may fail on some systems)
		if err := enterpriseBuf.BindToHardware(); err != nil {
			log.Printf("   Hardware Binding: Not available (%v)\n", err)
		} else {
			fmt.Printf("   Hardware Binding: SUCCESS\n")
			fmt.Printf("   Hardware Backed: %v\n", enterpriseBuf.IsHardwareBacked())
		}
	}

	// === 5. CRYPTOGRAPHIC OPERATIONS ===
	fmt.Println("\n5. Cryptographic Operations:")

	// Create a buffer with secret key material
	keyBuf, err := securebuf.NewWithFastEntropy(32)
	if err != nil {
		log.Printf("Key buffer error: %v", err)
	} else {
		defer keyBuf.Free()

		// Lock memory to prevent swapping
		if err := keyBuf.LockMemory(); err != nil {
			log.Printf("Memory lock error: %v", err)
		} else {
			fmt.Printf("   Memory Locked: %v\n", keyBuf.IsLocked())
		}

		// Compute HMAC of some data
		testData := []byte("Bitcoin Sprint Enterprise Security Test")
		hmacHex, err := keyBuf.HMACHex(testData)
		if err != nil {
			log.Printf("HMAC error: %v", err)
		} else {
			fmt.Printf("   HMAC-SHA256 (hex): %s\n", hmacHex[:32])
		}

		// Compute HMAC as base64url
		hmacB64, err := keyBuf.HMACBase64URL(testData)
		if err != nil {
			log.Printf("HMAC base64 error: %v", err)
		} else {
			fmt.Printf("   HMAC-SHA256 (base64url): %s\n", hmacB64[:32])
		}
	}

	// === 6. BITCOIN BLOOM FILTER ===
	fmt.Println("\n6. Bitcoin Bloom Filter:")

	// Create optimized Bitcoin Bloom filter
	bloomFilter, err := securebuf.NewBitcoinBloomFilterDefault()
	if err != nil {
		log.Printf("Bloom filter error: %v", err)
	} else {
		defer bloomFilter.Free()
		fmt.Printf("   Bitcoin Bloom Filter: CREATED\n")

		// Insert some mock UTXOs
		mockTxid1 := make([]byte, 32)
		mockTxid2 := make([]byte, 32)
		for i := range mockTxid1 {
			mockTxid1[i] = byte(i)
			mockTxid2[i] = byte(i + 64)
		}

		// Insert UTXOs
		if err := bloomFilter.InsertUTXO(mockTxid1, 0); err != nil {
			log.Printf("UTXO insert error: %v", err)
		} else {
			fmt.Printf("   Inserted UTXO: %s:0\n", hex.EncodeToString(mockTxid1[:8]))
		}

		// Check if UTXO exists
		exists, err := bloomFilter.ContainsUTXO(mockTxid1, 0)
		if err != nil {
			log.Printf("UTXO check error: %v", err)
		} else {
			fmt.Printf("   UTXO Exists: %v\n", exists)
		}

		// Check non-existent UTXO
		existsNot, err := bloomFilter.ContainsUTXO(mockTxid2, 1)
		if err != nil {
			log.Printf("UTXO check error: %v", err)
		} else {
			fmt.Printf("   Non-existent UTXO: %v\n", existsNot)
		}

		// Get bloom filter statistics
		stats, err := bloomFilter.GetStats()
		if err != nil {
			log.Printf("Bloom stats error: %v", err)
		} else {
			fmt.Printf("   Items: %d, FP Rate: %.6f, Memory: %d bytes\n",
				stats.ItemCount, stats.TheoreticalFPRate, stats.MemoryUsageBytes)
		}
	}

	// === 7. AUDIT AND COMPLIANCE ===
	fmt.Println("\n7. Audit and Compliance:")

	// Enable global audit logging
	if err := securebuf.EnableAuditLogging("/tmp/securebuffer_audit.log"); err != nil {
		log.Printf("Audit logging error: %v", err)
	} else {
		fmt.Printf("   Audit Logging: ENABLED\n")
		fmt.Printf("   Audit Status: %v\n", securebuf.IsAuditLoggingEnabled())

		// Set enterprise policy
		policyJSON := `{
			"max_buffer_lifetime": 3600,
			"require_memory_lock": true,
			"enable_tamper_detection": true,
			"audit_all_operations": true
		}`

		if err := securebuf.SetEnterprisePolicy(policyJSON); err != nil {
			log.Printf("Policy error: %v", err)
		} else {
			fmt.Printf("   Enterprise Policy: SET\n")
		}

		// Get compliance report
		report, err := securebuf.GetComplianceReport()
		if err != nil {
			log.Printf("Compliance report error: %v", err)
		} else {
			fmt.Printf("   Compliance Report: %d chars\n", len(report))
		}

		// Disable audit logging
		if err := securebuf.DisableAuditLogging(); err != nil {
			log.Printf("Disable audit error: %v", err)
		}
	}

	fmt.Println("\n✅ SecureBuffer Integration Demo Complete!")
}

// IntegrateWithBitcoinSprint shows how to use SecureBuffer in real Bitcoin Sprint components
func IntegrateWithBitcoinSprint() {
	fmt.Println("\n🚀 Bitcoin Sprint Integration Examples")
	fmt.Println("=====================================")

	// === 1. ENHANCED P2P AUTHENTICATION ===
	fmt.Println("\n1. Enhanced P2P Authentication:")

	// Create enterprise-grade secret buffer for HMAC
	secretBuf, err := entropy.CreateEnterpriseEntropyBuffer(64, securebuf.SecurityEnterprise)
	if err != nil {
		log.Printf("Secret buffer error: %v", err)
		return
	}
	defer secretBuf.Free()

	// Enable tamper detection and hardware binding
	if err := secretBuf.EnableTamperDetection(); err == nil {
		fmt.Printf("   P2P Secret: Tamper detection enabled\n")
	}

	if err := secretBuf.BindToHardware(); err == nil {
		fmt.Printf("   P2P Secret: Hardware-backed\n")
	}

	// === 2. API KEY GENERATION WITH BLOCKCHAIN ENTROPY ===
	fmt.Println("\n2. Enhanced API Key Generation:")

	// Use recent block headers for API key entropy
	mockRecentHeaders := [][]byte{
		make([]byte, 80), // Latest block
		make([]byte, 80), // Previous block
	}

	// Generate API key with blockchain entropy
	apiKeyBuf, err := entropy.CreateEntropyBufferWithHeaders(32, mockRecentHeaders)
	if err != nil {
		log.Printf("API key buffer error: %v", err)
	} else {
		defer apiKeyBuf.Free()

		// Lock memory and generate HMAC for API key
		if err := apiKeyBuf.LockMemory(); err == nil {
			apiKeyData := []byte("bitcoin-sprint-api-key-v2025")
			hmacKey, err := apiKeyBuf.HMACHex(apiKeyData)
			if err == nil {
				fmt.Printf("   Enhanced API Key: %s...\n", hmacKey[:16])
			}
		}
	}

	// === 3. UTXO BLOOM FILTER FOR MEMPOOL ===
	fmt.Println("\n3. High-Performance UTXO Filtering:")

	// Create optimized bloom filter for UTXO tracking
	utxoFilter, err := securebuf.NewBitcoinBloomFilter(
		1000000, // 1M bits
		7,       // 7 hash functions
		0x12345, // tweak
		0,       // flags
		3600,    // 1 hour max age
		1000,    // batch size
	)
	if err != nil {
		log.Printf("UTXO filter error: %v", err)
	} else {
		defer utxoFilter.Free()
		fmt.Printf("   UTXO Bloom Filter: Custom configuration ready\n")

		// Auto-cleanup old entries periodically
		if err := utxoFilter.AutoCleanup(); err == nil {
			fmt.Printf("   UTXO Filter: Auto-cleanup enabled\n")
		}
	}

	// === 4. ENTERPRISE COMPLIANCE INTEGRATION ===
	fmt.Println("\n4. Enterprise Compliance:")

	// Set up comprehensive audit logging
	auditPath := "/var/log/bitcoin-sprint/security-audit.log"
	if err := securebuf.EnableAuditLogging(auditPath); err == nil {
		fmt.Printf("   Audit Logging: %s\n", auditPath)

		// Set enterprise security policy
		enterprisePolicy := `{
			"security_level": "enterprise",
			"max_buffer_lifetime": 86400,
			"require_memory_lock": true,
			"enable_tamper_detection": true,
			"hardware_binding_required": false,
			"audit_all_operations": true,
			"side_channel_protection": true,
			"zero_copy_operations": true,
			"batch_crypto_operations": true
		}`

		if err := securebuf.SetEnterprisePolicy(enterprisePolicy); err == nil {
			fmt.Printf("   Enterprise Policy: Applied\n")
		}
	}

	fmt.Println("\n✅ Bitcoin Sprint Integration Examples Complete!")
}

func main() {
	DemonstrateSecureBufferIntegration()
	IntegrateWithBitcoinSprint()
}
