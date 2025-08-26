// SPDX-License-Identifier: MIT
// Bitcoin Sprint SecureBuffer (C FFI Header)

#ifndef SECUREBUFFER_H
#define SECUREBUFFER_H

#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C"
{
#endif

	typedef struct SecureBuffer SecureBuffer;

	SecureBuffer *securebuffer_new(size_t size);
	bool securebuffer_copy(SecureBuffer *sb, const unsigned char *src, size_t len);
	const unsigned char *securebuffer_data(SecureBuffer *sb);
	size_t securebuffer_len(const SecureBuffer *sb);
	void securebuffer_free(SecureBuffer *sb);
	bool securebuffer_self_check(void);

#ifdef __cplusplus
}
#endif

#endif
