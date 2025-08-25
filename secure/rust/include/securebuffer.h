// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer FFI Header

#ifndef SECUREBUFFER_H
#define SECUREBUFFER_H

#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct SecureBuffer SecureBuffer;

__declspec(dllexport) SecureBuffer* securebuffer_new(size_t size);
__declspec(dllexport) void securebuffer_free(SecureBuffer* buf);
__declspec(dllexport) bool securebuffer_copy(SecureBuffer* buf, const uint8_t* data, size_t len);
__declspec(dllexport) uint8_t* securebuffer_data(SecureBuffer* buf);
__declspec(dllexport) size_t securebuffer_len(SecureBuffer* buf);

#ifdef __cplusplus
}
#endif

#endif
