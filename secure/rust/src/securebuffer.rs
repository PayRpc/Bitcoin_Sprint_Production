// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer core

use std::sync::atomic::{AtomicUsize, Ordering};
use zeroize::Zeroize;
use std::io;
use thiserror::Error;

mod platform {
    #[cfg(unix)]
    pub fn lock_memory(ptr: *mut u8, len: usize) -> Result<(), std::io::Error> {
        unsafe {
            if libc::mlock(ptr as *mut libc::c_void, len) == 0 {
                Ok(())
            } else {
                Err(std::io::Error::last_os_error())
            }
        }
    }
    #[cfg(unix)]
    pub fn unlock_memory(ptr: *mut u8, len: usize) -> Result<(), std::io::Error> {
        unsafe {
            if libc::munlock(ptr as *mut libc::c_void, len) == 0 {
                Ok(())
            } else {
                Err(std::io::Error::last_os_error())
            }
        }
    }
    #[cfg(windows)]
    pub fn lock_memory(ptr: *mut u8, len: usize) -> Result<(), std::io::Error> {
        unsafe {
            if winapi::um::memoryapi::VirtualLock(ptr as *mut _, len) != 0 {
                Ok(())
            } else {
                Err(std::io::Error::last_os_error())
            }
        }
    }
    #[cfg(windows)]
    pub fn unlock_memory(ptr: *mut u8, len: usize) -> Result<(), std::io::Error> {
        unsafe {
            if winapi::um::memoryapi::VirtualUnlock(ptr as *mut _, len) != 0 {
                Ok(())
            } else {
                Err(std::io::Error::last_os_error())
            }
        }
    }
}

#[derive(Error, Debug)]
pub enum SecureBufferError {
    #[error("Invalid size")]
    InvalidSize,
    #[error("Allocation failed")]
    AllocationFailed,
    #[error("Lock failed: {0}")]
    LockFailed(#[source] io::Error),
    #[error("Copy overflow")]
    CopyOverflow,
    #[error("Invalid state")]
    InvalidState,
}

pub struct SecureBuffer {
    buffer: Vec<u8>,
    lock_count: AtomicUsize,
    is_valid: bool,
}

impl SecureBuffer {
    pub fn new(size: usize) -> Result<Self, SecureBufferError> {
        if size == 0 || size > (1 << 30) {
            return Err(SecureBufferError::InvalidSize);
        }
        let mut buffer = vec![0u8; size];
        platform::lock_memory(buffer.as_mut_ptr(), size)
            .map_err(SecureBufferError::LockFailed)?;
        Ok(Self { buffer, lock_count: AtomicUsize::new(1), is_valid: true })
    }
    pub fn copy_from_slice(&mut self, data: &[u8]) -> Result<(), SecureBufferError> {
        if !self.is_valid { return Err(SecureBufferError::InvalidState); }
        if data.len() > self.buffer.len() { return Err(SecureBufferError::CopyOverflow); }
        self.buffer[..data.len()].copy_from_slice(data);
        if data.len() < self.buffer.len() {
            self.buffer[data.len()..].zeroize();
        }
        Ok(())
    }
    pub fn as_slice(&self) -> Option<&[u8]> {
        if !self.is_valid { None } else { Some(&self.buffer) }
    }
    pub fn as_mut_slice(&mut self) -> Option<&mut [u8]> {
        if !self.is_valid { None } else { Some(&mut self.buffer) }
    }
    pub fn len(&self) -> usize { self.buffer.len() }
}

impl Zeroize for SecureBuffer {
    fn zeroize(&mut self) {
        self.buffer.zeroize();
    }
}

impl Drop for SecureBuffer {
    fn drop(&mut self) {
        self.is_valid = false;
        self.buffer.zeroize();
        if self.lock_count.load(Ordering::Acquire) > 0 {
            let _ = platform::unlock_memory(self.buffer.as_mut_ptr(), self.buffer.len());
        }
    }
}
