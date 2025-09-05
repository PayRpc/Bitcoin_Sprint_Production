# TurboValidator

High-performance block/transaction validator for Bitcoin Sprint with PQC mix-in, entropy weighting, and enterprise audit features.

## Features
- PQC mix-in (Kyber/Dilithium) policy
- entropy_pqc_weight metric
- Receipt/proof bundle for `/entropy/hybrid`
- JSON serialization for audit
- Unit tests for all features

## Usage
Add as a Rust crate and use `TurboValidator` for block/tx validation and entropy audit receipts.
