// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Enterprise SecureBuffer FFI Header
// Comprehensive memory protection and cryptographic operations

#ifndef SECUREBUFFER_H
#define SECUREBUFFER_H

#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>

// Version information
#define SECUREBUFFER_VERSION_MAJOR 2
#define SECUREBUFFER_VERSION_MINOR 1
#define SECUREBUFFER_VERSION_PATCH 0
#define SECUREBUFFER_VERSION_STRING "2.1.0"

// Cross-platform API export macro
#if defined(_WIN32) || defined(_WIN64)
#define SECUREBUFFER_API __declspec(dllexport)
#elif defined(__GNUC__) || defined(__clang__)
#define SECUREBUFFER_API __attribute__((visibility("default")))
#else
#define SECUREBUFFER_API
#endif

// Error codes
typedef enum
{
	SECUREBUFFER_SUCCESS = 0,
	SECUREBUFFER_ERROR_NULL_POINTER = -1,
	SECUREBUFFER_ERROR_INVALID_SIZE = -2,
	SECUREBUFFER_ERROR_ALLOCATION_FAILED = -3,
	SECUREBUFFER_ERROR_BUFFER_OVERFLOW = -4,
	SECUREBUFFER_ERROR_INTEGRITY_CHECK_FAILED = -5,
	SECUREBUFFER_ERROR_CRYPTO_OPERATION_FAILED = -6,
	SECUREBUFFER_ERROR_THREAD_SAFETY_VIOLATION = -7
} SecureBufferError;

// Security levels
typedef enum
{
	SECUREBUFFER_SECURITY_STANDARD = 0,
	SECUREBUFFER_SECURITY_HIGH = 1,
	SECUREBUFFER_SECURITY_ENTERPRISE = 2,
	SECUREBUFFER_SECURITY_FORENSIC_RESISTANT = 3
} SecureBufferSecurityLevel;

// Hash algorithms
typedef enum
{
	SECUREBUFFER_HASH_SHA256 = 0,
	SECUREBUFFER_HASH_SHA512 = 1,
	SECUREBUFFER_HASH_BLAKE3 = 2
} SecureBufferHashAlgorithm;

// Metrics structure
typedef struct
{
	uint64_t total_allocations;
	uint64_t total_deallocations;
	uint64_t current_active_buffers;
	uint64_t peak_active_buffers;
	uint64_t total_bytes_allocated;
	uint64_t total_bytes_deallocated;
	uint64_t integrity_checks_performed;
	uint64_t integrity_check_failures;
	double average_operation_time_ns;
	uint64_t crypto_operations_count;
} SecureBufferMetrics;

#ifdef __cplusplus
extern "C"
{
#endif

	// Core types
	typedef struct SecureBuffer SecureBuffer;
	typedef struct SecureChannelPool SecureChannelPool;

	// === Core Buffer Operations ===
	SECUREBUFFER_API SecureBuffer *securebuffer_new(size_t size);
	SECUREBUFFER_API SecureBuffer *securebuffer_new_with_security_level(size_t size, SecureBufferSecurityLevel level);
	SECUREBUFFER_API void securebuffer_free(SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_copy(SecureBuffer *buf, const uint8_t *data, size_t len);
	SECUREBUFFER_API uint8_t *securebuffer_data(SecureBuffer *buf);
	SECUREBUFFER_API const uint8_t *securebuffer_data_readonly(const SecureBuffer *buf);
	SECUREBUFFER_API size_t securebuffer_len(const SecureBuffer *buf);
	SECUREBUFFER_API size_t securebuffer_capacity(const SecureBuffer *buf);

	// === Memory Protection ===
	SECUREBUFFER_API SecureBufferError securebuffer_lock_memory(SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_unlock_memory(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_is_locked(const SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_zero_memory(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_integrity_check(const SecureBuffer *buf);

	// === Cryptographic Operations ===
	SECUREBUFFER_API char *securebuffer_hmac_hex(SecureBuffer *buf, const uint8_t *data, size_t data_len);
	SECUREBUFFER_API char *securebuffer_hmac_base64url(SecureBuffer *buf, const uint8_t *data, size_t data_len);
	SECUREBUFFER_API char *securebuffer_hmac_with_algorithm(SecureBuffer *buf, const uint8_t *data, size_t data_len, SecureBufferHashAlgorithm algo);
	SECUREBUFFER_API SecureBufferError securebuffer_derive_key(SecureBuffer *buf, const uint8_t *password, size_t password_len, const uint8_t *salt, size_t salt_len, uint32_t iterations);
	SECUREBUFFER_API SecureBufferError securebuffer_encrypt_aes256_gcm(SecureBuffer *buf, const uint8_t *key, const uint8_t *nonce, SecureBuffer *output);
	SECUREBUFFER_API SecureBufferError securebuffer_decrypt_aes256_gcm(SecureBuffer *buf, const uint8_t *key, const uint8_t *nonce, SecureBuffer *output);

	// === Thread Safety ===
	SECUREBUFFER_API SecureBufferError securebuffer_acquire_read_lock(SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_acquire_write_lock(SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_release_lock(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_is_thread_safe(const SecureBuffer *buf);

	// === SecureChannelPool Operations ===
	SECUREBUFFER_API SecureChannelPool *securechannel_pool_new(size_t max_connections, const char *endpoint);
	SECUREBUFFER_API void securechannel_pool_free(SecureChannelPool *pool);
	SECUREBUFFER_API SecureBufferError securechannel_pool_send(SecureChannelPool *pool, const uint8_t *data, size_t len, SecureBuffer *response);
	SECUREBUFFER_API bool securechannel_pool_is_healthy(const SecureChannelPool *pool);
	SECUREBUFFER_API char *securechannel_pool_get_status_json(const SecureChannelPool *pool);
	SECUREBUFFER_API double securechannel_pool_get_health_score(const SecureChannelPool *pool);

	// === Metrics and Monitoring ===
	SECUREBUFFER_API SecureBufferMetrics securebuffer_get_global_metrics(void);
	SECUREBUFFER_API char *securebuffer_get_metrics_json(void);
	SECUREBUFFER_API void securebuffer_reset_metrics(void);
	SECUREBUFFER_API char *securebuffer_get_prometheus_metrics(void);

	// === Utility Functions ===
	SECUREBUFFER_API void securebuffer_free_cstr(char *s);
	SECUREBUFFER_API bool securebuffer_self_check(void);
	SECUREBUFFER_API char *securebuffer_get_version_info(void);
	SECUREBUFFER_API bool securebuffer_is_enterprise_build(void);
	SECUREBUFFER_API char *securebuffer_get_build_info(void);

	// === Enterprise Features ===
	SECUREBUFFER_API SecureBufferError securebuffer_enable_audit_logging(const char *log_path);
	SECUREBUFFER_API SecureBufferError securebuffer_disable_audit_logging(void);
	SECUREBUFFER_API bool securebuffer_is_audit_logging_enabled(void);
	SECUREBUFFER_API char *securebuffer_get_compliance_report(void);
	SECUREBUFFER_API SecureBufferError securebuffer_set_enterprise_policy(const char *policy_json);

	// === Error Handling ===
	SECUREBUFFER_API const char *securebuffer_error_string(SecureBufferError error);
	SECUREBUFFER_API SecureBufferError securebuffer_get_last_error(void);
	SECUREBUFFER_API void securebuffer_clear_last_error(void);

#ifdef __cplusplus
}
#endif

#endif // SECUREBUFFER_H
