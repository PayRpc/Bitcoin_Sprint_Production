//! TurboValidator: High-performance block/transaction validator for Bitcoin Sprint
//! Extend with custom rules, cryptographic checks, and anti-fraud logic as needed.

use serde::{Deserialize, Serialize};
use serde_json;
use std::error::Error;
use std::fmt;

/// Validation errors for blocks/transactions
#[derive(Debug)]
pub enum ValidationError {
    InvalidBlock(String),
    InvalidTransaction(String),
    SignatureError(String),
    DoubleSpend(String),
    Other(String),
}

impl fmt::Display for ValidationError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ValidationError::InvalidBlock(msg) => write!(f, "Invalid block: {}", msg),
            ValidationError::InvalidTransaction(msg) => write!(f, "Invalid transaction: {}", msg),
            ValidationError::SignatureError(msg) => write!(f, "Signature error: {}", msg),
            ValidationError::DoubleSpend(msg) => write!(f, "Double spend: {}", msg),
            ValidationError::Other(msg) => write!(f, "Validation error: {}", msg),
        }
    }
}

impl Error for ValidationError {}

/// Policy for PQC mix-in weighting and controls
#[derive(Debug, Clone)]
pub struct PQCPolicy {
    /// Enable Kyber PQC verification
    pub kyber_enabled: bool,
    /// Enable Dilithium PQC verification
    pub dilithium_enabled: bool,
    /// Entropy PQC weight (0.0..1.0)
    pub entropy_pqc_weight: f64,
}

impl Default for PQCPolicy {
    fn default() -> Self {
        Self {
            kyber_enabled: true,
            dilithium_enabled: true,
            entropy_pqc_weight: 0.5,
        }
    }
}

/// TurboValidator struct: stateless, thread-safe, with PQC policy
#[derive(Debug, Clone)]
pub struct TurboValidator {
    pub pqc_policy: PQCPolicy,
}

impl Default for TurboValidator {
    fn default() -> Self {
        Self {
            pqc_policy: PQCPolicy::default(),
        }
    }
}

impl TurboValidator {
    /// Validate a block (stub: extend with real logic)
    pub fn validate_block(&self, block: &[u8]) -> Result<(), ValidationError> {
        if block.is_empty() {
            return Err(ValidationError::InvalidBlock("Block data is empty".into()));
        }
        // PQC mix-in: simulate Kyber/Dilithium checks
        if self.pqc_policy.kyber_enabled {
            // TODO: Call Kyber verification (stub)
        }
        if self.pqc_policy.dilithium_enabled {
            // TODO: Call Dilithium verification (stub)
        }
        Ok(())
    }

    /// Validate a transaction (stub: extend with real logic)
    pub fn validate_transaction(&self, tx: &[u8]) -> Result<(), ValidationError> {
        if tx.is_empty() {
            return Err(ValidationError::InvalidTransaction("Transaction data is empty".into()));
        }
        // PQC mix-in: simulate Kyber/Dilithium checks
        if self.pqc_policy.kyber_enabled {
            // TODO: Call Kyber verification (stub)
        }
        if self.pqc_policy.dilithium_enabled {
            // TODO: Call Dilithium verification (stub)
        }
        Ok(())
    }

    /// Get current entropy_pqc_weight metric
    pub fn entropy_pqc_weight(&self) -> f64 {
        self.pqc_policy.entropy_pqc_weight
    }

    /// Set PQC policy (for ops control)
    pub fn set_pqc_policy(&mut self, policy: PQCPolicy) {
        self.pqc_policy = policy;
    }

    /// Generate a receipt + proof bundle for /entropy/hybrid
    pub fn generate_entropy_hybrid_receipt(
        &self,
        beacon_round: u64,
        attestation: &str,
        proof_hash: &str,
        verifier_id: &str,
    ) -> EntropyHybridReceipt {
        EntropyHybridReceipt {
            beacon_round,
            attestation: attestation.to_string(),
            proof_hash: proof_hash.to_string(),
            verifier_id: verifier_id.to_string(),
            pqc_weight: self.entropy_pqc_weight(),
        }
    }

    /// Serialize receipt to JSON for enterprise audit
    pub fn serialize_receipt_json(receipt: &EntropyHybridReceipt) -> Result<String, serde_json::Error> {
        serde_json::to_string(receipt)
    }
}

/// Receipt + proof bundle for /entropy/hybrid
#[derive(Debug, Serialize, Deserialize)]
pub struct EntropyHybridReceipt {
    pub beacon_round: u64,
    pub attestation: String,
    pub proof_hash: String,
    pub verifier_id: String,
    pub pqc_weight: f64,
}

#[cfg(test)]
mod pqc_tests {
    use super::*;
    #[test]
    fn test_entropy_pqc_weight() {
        let validator = TurboValidator::default();
        assert_eq!(validator.entropy_pqc_weight(), 0.5);
    }
    #[test]
    fn test_receipt_json() {
        let validator = TurboValidator::default();
        let receipt = validator.generate_entropy_hybrid_receipt(42, "attest", "proofhash", "verifierX");
        let json = TurboValidator::serialize_receipt_json(&receipt).unwrap();
        assert!(json.contains("beacon_round"));
        assert!(json.contains("verifierX"));
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_empty_block() {
        let validator = TurboValidator::default();
        assert!(validator.validate_block(&[]).is_err());
    }

    #[test]
    fn test_empty_tx() {
        let validator = TurboValidator::default();
        assert!(validator.validate_transaction(&[]).is_err());
    }
}
