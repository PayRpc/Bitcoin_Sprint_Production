// lib.rs
// Bitcoin Sprint Storage Verifier Library
// Exposes the netkit networking utilities

pub mod netkit;

// Re-export key functions for easy access
pub use netkit::{
    connect_happy,
    connect_tls,
    connect_tuned,
    read_exact_deadline,
    write_all_deadline,
    pad_frame,
    TlsStream,
    tls_connector,
};
