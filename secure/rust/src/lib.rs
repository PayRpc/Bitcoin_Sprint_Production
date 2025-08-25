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
