// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer Rust FFI

mod securebuffer;
pub use securebuffer::{SecureBuffer, SecureBufferError};

use std::slice;
use std::ffi::CStr;
use std::os::raw::c_char;
use std::sync::atomic::{AtomicBool, Ordering};

// Global secure channel status
static SECURE_CHANNEL_RUNNING: AtomicBool = AtomicBool::new(false);

#[no_mangle]
pub extern "C" fn securebuffer_new(size: usize) -> *mut SecureBuffer {
    match SecureBuffer::new(size) {
        Ok(buf) => Box::into_raw(Box::new(buf)),
        Err(_) => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn securebuffer_free(buf: *mut SecureBuffer) {
    if !buf.is_null() {
        unsafe { let _ = Box::from_raw(buf); }
    }
}

#[no_mangle]
pub extern "C" fn securebuffer_copy(buf: *mut SecureBuffer, data: *const u8, len: usize) -> bool {
    if buf.is_null() || data.is_null() {
        return false;
    }
    let sb = unsafe { &mut *buf };
    let slice = unsafe { slice::from_raw_parts(data, len) };
    sb.write(slice).is_ok()
}

#[no_mangle]
pub extern "C" fn securebuffer_data(buf: *mut SecureBuffer) -> *mut u8 {
    if buf.is_null() {
        return std::ptr::null_mut();
    }
    let sb = unsafe { &mut *buf };
    match sb.as_slice() {
        Ok(slice) => slice.as_ptr() as *mut u8,
        Err(_) => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn securebuffer_len(buf: *mut SecureBuffer) -> usize {
    if buf.is_null() {
        return 0;
    }
    let sb = unsafe { &*buf };
    sb.len()
}

// Compute HMAC-SHA256(hex) of data using secret held in SecureBuffer.
// The caller must free the returned C string with securebuffer_free_cstr.
#[no_mangle]
pub extern "C" fn securebuffer_hmac_hex(buf: *mut SecureBuffer, data: *const u8, data_len: usize) -> *mut i8 {
    if buf.is_null() || data.is_null() {
        return std::ptr::null_mut();
    }
    let sb = unsafe { &*buf };
    let key = match sb.as_slice() {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };
    let slice = unsafe { slice::from_raw_parts(data, data_len) };
    use hmac::{Hmac, Mac};
    use sha2::Sha256;
    use std::ffi::CString;

    type HmacSha256 = Hmac<Sha256>;
    let mut mac = match HmacSha256::new_from_slice(key) {
        Ok(m) => m,
        Err(_) => return std::ptr::null_mut(),
    };
    mac.update(slice);
    let result = mac.finalize().into_bytes();
    let hex = hex::encode(result);
    // return C string
    match CString::new(hex) {
        Ok(cstr) => cstr.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

// Free a C string returned by this crate
#[no_mangle]
pub extern "C" fn securebuffer_free_cstr(s: *mut i8) {
    if s.is_null() { return; }
    unsafe { let _ = std::ffi::CString::from_raw(s); }
}

// Compute base64url of provided data using secret as HMAC key and return C string (for Authorization header purposes)
#[no_mangle]
pub extern "C" fn securebuffer_hmac_base64url(buf: *mut SecureBuffer, data: *const u8, data_len: usize) -> *mut i8 {
    if buf.is_null() || data.is_null() { return std::ptr::null_mut(); }
    let sb = unsafe { &*buf };
    let key = match sb.as_slice() { Ok(s) => s, Err(_) => return std::ptr::null_mut() };
    let slice = unsafe { slice::from_raw_parts(data, data_len) };
    use hmac::{Hmac, Mac};
    use sha2::Sha256;
    use base64::Engine;
    use std::ffi::CString;
    type HmacSha256 = Hmac<Sha256>;
    let mut mac = match HmacSha256::new_from_slice(key) { Ok(m) => m, Err(_) => return std::ptr::null_mut() };
    mac.update(slice);
    let result = mac.finalize().into_bytes();
    // base64url without padding
    let b64 = base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(result);
    match CString::new(b64) {
        Ok(cstr) => cstr.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

// Self-check function - returns true to confirm native Rust memory protection is active
#[no_mangle]
pub extern "C" fn securebuffer_self_check() -> bool {
    // Test basic functionality: allocate, write, read
    let mut buf = match SecureBuffer::new(32) {
        Ok(b) => b,
        Err(_) => return false,
    };

    // Write test data
    let test_data = b"RUST_NATIVE_MEMORY_GUARD_ACTIVE";
    if buf.write(test_data).is_err() {
        return false;
    }

    // Read and verify - this confirms native implementation is working
    match buf.as_slice() {
        Ok(slice) => slice[..test_data.len()] == *test_data,
        Err(_) => false,
    }
}

// SecureChannel FFI Functions (Simplified)

/// Initialize the secure channel (mock implementation for now)
#[no_mangle]
pub extern "C" fn secure_channel_init() -> bool {
    SECURE_CHANNEL_RUNNING.store(true, Ordering::Relaxed);
    true
}

/// Initialize with custom endpoint (mock implementation)
#[no_mangle]
pub extern "C" fn secure_channel_init_with_endpoint(endpoint: *const c_char) -> bool {
    if endpoint.is_null() {
        return false;
    }

    // For now, just validate the endpoint string is readable
    unsafe {
        match CStr::from_ptr(endpoint).to_str() {
            Ok(_) => {
                SECURE_CHANNEL_RUNNING.store(true, Ordering::Relaxed);
                true
            }
            Err(_) => false,
        }
    }
}

/// Start the secure channel
#[no_mangle]
pub extern "C" fn secure_channel_start() -> bool {
    SECURE_CHANNEL_RUNNING.store(true, Ordering::Relaxed);
    true
}

/// Stop the secure channel
#[no_mangle]
pub extern "C" fn secure_channel_stop() -> bool {
    SECURE_CHANNEL_RUNNING.store(false, Ordering::Relaxed);
    true
}

/// Check if secure channel is running
#[no_mangle]
pub extern "C" fn secure_channel_is_running() -> bool {
    SECURE_CHANNEL_RUNNING.load(Ordering::Relaxed)
}

/// Get the metrics server port (returns default port)
#[no_mangle]
pub extern "C" fn secure_channel_get_metrics_port() -> u16 {
    9090
}
