// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer Rust FFI

mod securebuffer;
pub use securebuffer::{SecureBuffer, SecureBufferError};

use std::slice;

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
    sb.copy_from_slice(slice).is_ok()
}

#[no_mangle]
pub extern "C" fn securebuffer_data(buf: *mut SecureBuffer) -> *mut u8 {
    if buf.is_null() {
        return std::ptr::null_mut();
    }
    let sb = unsafe { &mut *buf };
    match sb.as_mut_slice() {
        Some(slice) => slice.as_mut_ptr(),
        None => std::ptr::null_mut(),
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
        Some(s) => s,
        None => return std::ptr::null_mut(),
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
    let key = match sb.as_slice() { Some(s) => s, None => return std::ptr::null_mut() };
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

// Self-check function to verify SecureBuffer functionality
#[no_mangle]
pub extern "C" fn securebuffer_self_check() -> bool {
    // Allocate a small buffer
    let mut buf = match SecureBuffer::new(32) {
        Ok(b) => b,
        Err(_) => return false,
    };

    // Write test data
    let test_data = b"self-check memory test data";
    if buf.copy_from_slice(test_data).is_err() {
        return false;
    }

    // Read the data back to verify it was written
    let before = match buf.as_slice() {
        Some(slice) => slice[..test_data.len()].to_vec(),
        None => return false,
    };

    // Verify the data matches
    if before != test_data {
        return false;
    }

    // Zeroize explicitly
    buf.zeroize();

    // After zeroize, all bytes must be 0
    let after = match buf.as_slice() {
        Some(slice) => slice[..test_data.len()].to_vec(),
        None => return false,
    };

    // Verify data was zeroized and that it's different from before
    before != after && after.iter().all(|&b| b == 0)
}
