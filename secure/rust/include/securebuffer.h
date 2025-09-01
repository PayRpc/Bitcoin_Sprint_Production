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

// Enterprise configuration constants
#define SECUREBUFFER_MAX_BUFFER_LIFETIME_DEFAULT 86400 // 24 hours in seconds
#define SECUREBUFFER_ZEROIZATION_INTERVAL_DEFAULT 3600 // 1 hour in seconds
#define SECUREBUFFER_HARDWARE_TIMEOUT_MS 5000		   // 5 seconds for hardware operations
#define SECUREBUFFER_BATCH_MAX_SIZE 1024			   // Maximum batch operation size
#define SECUREBUFFER_UUID_LENGTH 37					   // UUID string length including null terminator

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
	SECUREBUFFER_ERROR_THREAD_SAFETY_VIOLATION = -7,
	SECUREBUFFER_ERROR_HARDWARE_NOT_AVAILABLE = -8,
	SECUREBUFFER_ERROR_TAMPER_DETECTED = -9,
	SECUREBUFFER_ERROR_POLICY_VIOLATION = -10,
	SECUREBUFFER_ERROR_EXPIRED = -11,
	SECUREBUFFER_ERROR_SIDE_CHANNEL_ATTACK = -12,
	SECUREBUFFER_ERROR_ZERO_COPY_FAILED = -13,
	SECUREBUFFER_ERROR_BATCH_OPERATION_FAILED = -14
} SecureBufferError;

// Security levels
typedef enum
{
	SECUREBUFFER_SECURITY_STANDARD = 0,
	SECUREBUFFER_SECURITY_HIGH = 1,
	SECUREBUFFER_SECURITY_ENTERPRISE = 2,
	SECUREBUFFER_SECURITY_FORENSIC_RESISTANT = 3,
	SECUREBUFFER_SECURITY_HARDWARE = 4 // TPM/HSM/SGX integration
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
	uint64_t hardware_operations_count;
	uint64_t batch_operations_count;
	uint64_t zero_copy_operations_count;
	uint64_t tamper_detection_events;
	uint64_t side_channel_protection_activations;
} SecureBufferMetrics;

#ifdef __cplusplus
extern "C"
{
#endif

	// Core types
	typedef struct SecureBuffer SecureBuffer;
	typedef struct SecureBuffer *SecureBufferHandle;
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
	SECUREBUFFER_API SecureBufferError securebuffer_rotate_key(SecureBuffer *buf);

	// === Hardware-backed Security ===
	SECUREBUFFER_API SecureBufferError securebuffer_bind_to_hardware(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_is_hardware_backed(const SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_enable_side_channel_protection(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_constant_time_compare(const SecureBuffer *buf1, const SecureBuffer *buf2);

// === Zero-copy IPC ===
#if defined(__unix__) || defined(__unix) || defined(__APPLE__)
	SECUREBUFFER_API int securebuffer_as_fd(const SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_share_with_process(SecureBuffer *buf, int pid);
#endif

	// === Batch Crypto Operations ===
	SECUREBUFFER_API char **securebuffer_hmac_batch(
		SecureBuffer *buf,
		const uint8_t **data_list,
		size_t *data_lens,
		size_t count);
	SECUREBUFFER_API void securebuffer_free_batch_results(char **results, size_t count);

	// === Thread Safety ===
	SECUREBUFFER_API SecureBufferError securebuffer_acquire_read_lock(SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_acquire_write_lock(SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_release_lock(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_is_thread_safe(const SecureBuffer *buf);

	// === Metadata and Compliance ===
	SECUREBUFFER_API char *securebuffer_get_uuid(const SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_verify_metadata(const SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_set_max_lifetime(SecureBuffer *buf, uint64_t max_lifetime_seconds);
	SECUREBUFFER_API uint64_t securebuffer_get_creation_timestamp(const SecureBuffer *buf);
	SECUREBUFFER_API uint64_t securebuffer_get_last_access_timestamp(const SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_is_expired(const SecureBuffer *buf);

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

	// === Advanced Enterprise Features ===
	SECUREBUFFER_API SecureBufferError securebuffer_enable_tamper_detection(SecureBuffer *buf);
	SECUREBUFFER_API bool securebuffer_is_tampered(const SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_force_zeroization_schedule(SecureBuffer *buf, uint64_t interval_seconds);
	SECUREBUFFER_API char *securebuffer_get_security_audit_log(const SecureBuffer *buf);
	SECUREBUFFER_API SecureBufferError securebuffer_validate_policy_compliance(const SecureBuffer *buf);

	// === Performance Optimizations ===
	SECUREBUFFER_API bool securebuffer_has_hardware_acceleration(void);
	SECUREBUFFER_API char *securebuffer_get_acceleration_info(void);
	SECUREBUFFER_API SecureBufferError securebuffer_prefault_pages(SecureBuffer *buf);
	SECUREBUFFER_API double securebuffer_benchmark_operations(size_t buffer_size, size_t iterations);

	// === Enterprise Features ===
	SECUREBUFFER_API SecureBufferError securebuffer_enable_audit_logging(const char *log_path);
	SECUREBUFFER_API SecureBufferError securebuffer_disable_audit_logging(void);
	SECUREBUFFER_API bool securebuffer_is_audit_logging_enabled(void);
	SECUREBUFFER_API char *securebuffer_get_compliance_report(void);
	SECUREBUFFER_API SecureBufferError securebuffer_set_enterprise_policy(const char *policy_json);

	// === Entropy Integration ===
	// Fill existing buffer with fast entropy (OS RNG + timing jitter)
	SECUREBUFFER_API int securebuffer_fill_fast_entropy(void *buffer);

	// Fill existing buffer with hybrid entropy (OS RNG + Bitcoin headers + jitter)
	SECUREBUFFER_API int securebuffer_fill_hybrid_entropy(
		void *buffer,
		const uint8_t *headers_ptr,
		size_t headers_len,
		size_t header_count);

	// Fill existing buffer with enterprise-grade entropy
	SECUREBUFFER_API int securebuffer_fill_enterprise_entropy(
		void *buffer,
		const uint8_t *headers_ptr,
		size_t headers_len,
		size_t header_count,
		const uint8_t *additional_data_ptr,
		size_t additional_data_len);

	// Create new buffer pre-filled with fast entropy
	SECUREBUFFER_API void *securebuffer_new_with_fast_entropy(size_t capacity);

	// Create new buffer pre-filled with hybrid entropy
	SECUREBUFFER_API void *securebuffer_new_with_hybrid_entropy(
		size_t capacity,
		const uint8_t *headers_ptr,
		size_t headers_len,
		size_t header_count);

	// Refresh buffer contents with new entropy
	SECUREBUFFER_API int securebuffer_refresh_entropy(void *buffer);

	// Mix additional entropy into existing buffer content
	SECUREBUFFER_API int securebuffer_mix_entropy(
		void *buffer,
		const uint8_t *headers_ptr,
		size_t headers_len,
		size_t header_count);

	// === Bitcoin Bloom Filter API ===
	// Create new Bitcoin Bloom Filter with optimized configuration
	SECUREBUFFER_API void *bitcoin_bloom_filter_new(
		size_t size_bits,
		uint8_t num_hashes,
		uint32_t tweak,
		uint8_t flags,
		uint64_t max_age_seconds,
		size_t batch_size);

	// Create Bitcoin Bloom Filter with default configuration
	SECUREBUFFER_API void *bitcoin_bloom_filter_new_default(void);

	// Destroy Bitcoin Bloom Filter and securely zeroize memory
	SECUREBUFFER_API void bitcoin_bloom_filter_destroy(void *filter);

	// Insert single UTXO into bloom filter
	SECUREBUFFER_API int bitcoin_bloom_filter_insert_utxo(
		void *filter,
		const uint8_t *txid_bytes,
		uint32_t vout);

	// Insert batch of UTXOs into bloom filter (maximum performance)
	SECUREBUFFER_API int bitcoin_bloom_filter_insert_batch(
		void *filter,
		const uint8_t *txid_bytes,
		const uint32_t *vouts,
		size_t count);

	// Check if single UTXO exists in bloom filter
	SECUREBUFFER_API int bitcoin_bloom_filter_contains_utxo(
		void *filter,
		const uint8_t *txid_bytes,
		uint32_t vout);

	// Check batch of UTXOs in bloom filter
	SECUREBUFFER_API int bitcoin_bloom_filter_contains_batch(
		void *filter,
		const uint8_t *txid_bytes,
		const uint32_t *vouts,
		size_t count,
		bool *results);

	// Load entire Bitcoin block into bloom filter
	SECUREBUFFER_API int bitcoin_bloom_filter_load_block(
		void *filter,
		const uint8_t *block_data,
		size_t block_size);

	// Get bloom filter statistics
	SECUREBUFFER_API int bitcoin_bloom_filter_get_stats(
		void *filter,
		uint64_t *item_count,
		uint64_t *false_positive_count,
		double *theoretical_fp_rate,
		size_t *memory_usage_bytes,
		size_t *timestamp_entries,
		double *average_age_seconds);

	// Get theoretical false positive rate
	SECUREBUFFER_API double bitcoin_bloom_filter_false_positive_rate(void *filter);

	// Cleanup old entries to maintain performance
	SECUREBUFFER_API int bitcoin_bloom_filter_cleanup(void *filter);

	// Auto-cleanup if needed (call periodically)
	SECUREBUFFER_API int bitcoin_bloom_filter_auto_cleanup(void *filter);

	// === Direct Entropy Functions ===
	SECUREBUFFER_API int fast_entropy_c(unsigned char *output);
	SECUREBUFFER_API int hybrid_entropy_c(
		const unsigned char **headers,
		const size_t *header_lengths,
		size_t header_count,
		unsigned char *output
	);
	SECUREBUFFER_API int enterprise_entropy_c(
		const unsigned char **headers,
		const size_t *header_lengths,
		size_t header_count,
		const unsigned char *additional_data,
		size_t additional_data_len,
		unsigned char *output
	);
	SECUREBUFFER_API int system_fingerprint_c(unsigned char *output);
	SECUREBUFFER_API float get_cpu_temperature_c(void);
	SECUREBUFFER_API int fast_entropy_with_fingerprint_c(unsigned char *output);
	SECUREBUFFER_API int hybrid_entropy_with_fingerprint_c(
		const unsigned char **headers,
		const size_t *header_lengths,
		size_t header_count,
		unsigned char *output
	);

	// === Error Handling ===
	SECUREBUFFER_API const char *securebuffer_error_string(SecureBufferError error);
	SECUREBUFFER_API SecureBufferError securebuffer_get_last_error(void);
	SECUREBUFFER_API void securebuffer_clear_last_error(void);

#ifdef __cplusplus
}
#endif

#endif // SECUREBUFFER_H
