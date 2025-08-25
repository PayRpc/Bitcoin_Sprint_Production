// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer FFI Header

#ifndef SECUREBUFFER_H
#define SECUREBUFFER_H

#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>

// Cross-platform API export macro
#if defined(_WIN32) || defined(_WIN64)
#define SECUREBUFFER_API __declspec(dllexport)
#elif defined(__GNUC__) || defined(__clang__)
#define SECUREBUFFER_API __attribute__((visibility("default")))
#else
#define SECUREBUFFER_API
#endif

#ifdef __cplusplus
extern "C"
{
#endif

	typedef struct SecureBuffer SecureBuffer;

	SECUREBUFFER_API SecureBuffer *securebuffer_new(size_t size);
	SECUREBUFFER_API void securebuffer_free(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_copy(SecureBuffer *buf, const uint8_t *data, size_t len);
	SECUREBUFFER_API uint8_t *securebuffer_data(SecureBuffer *buf);
	SECUREBUFFER_API size_t securebuffer_len(SecureBuffer *buf);
	SECUREBUFFER_API char *securebuffer_hmac_hex(SecureBuffer *buf, const uint8_t *data, size_t data_len);
	SECUREBUFFER_API char *securebuffer_hmac_base64url(SecureBuffer *buf, const uint8_t *data, size_t data_len);
	SECUREBUFFER_API void securebuffer_free_cstr(char *s);

#ifdef __cplusplus
}
#endif

#endif
