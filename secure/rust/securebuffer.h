// SPDX-License-Identifier: MIT
// Bitcoin Sprint SecureBuffer (C FFI Header)

#ifndef SECUREBUFFER_H
#define SECUREBUFFER_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct SecureBuffer SecureBuffer;

SecureBuffer* securebuffer_new(size_t size);
int securebuffer_copy(SecureBuffer* sb, const char* src);
size_t securebuffer_len(const SecureBuffer* sb);
void securebuffer_free(SecureBuffer* sb);

#ifdef __cplusplus
}
#endif

#endif
