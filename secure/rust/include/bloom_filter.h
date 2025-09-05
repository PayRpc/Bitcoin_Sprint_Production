// SPDX-License-Identifier: MIT
// Universal Sprint Bloom Filter - Production Quality C Header
// Provides FFI for Go and other languages

#ifndef BLOOM_FILTER_H
#define BLOOM_FILTER_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// Opaque type for UniversalBloomFilter
typedef struct UniversalBloomFilter UniversalBloomFilter;

// Configuration struct for Bloom Filter
typedef struct {
    const char* network;
    uint64_t size;
    uint8_t num_hashes;
    uint32_t tweak;
    uint8_t flags;
    uint64_t max_age_seconds;
    uint64_t batch_size;
    bool enable_compression;
    bool enable_metrics;
} BloomConfig;

// Error codes
typedef enum {
    BLOOM_OK = 0,
    BLOOM_ERR_INVALID_CONFIG = 1,
    BLOOM_ERR_INVALID_INPUT = 2,
    BLOOM_ERR_HASH_ERROR = 3,
    BLOOM_ERR_MEMORY = 4,
    BLOOM_ERR_CONCURRENCY = 5
} BloomFilterErrorCode;

// Create a new Bloom Filter
UniversalBloomFilter* bloom_filter_new(const BloomConfig* config, BloomFilterErrorCode* err);

// Destroy Bloom Filter
void bloom_filter_free(UniversalBloomFilter* filter);

// Insert data into Bloom Filter
bool bloom_filter_insert(UniversalBloomFilter* filter, const uint8_t* data, uint64_t len);

// Check if data is present
bool bloom_filter_contains(const UniversalBloomFilter* filter, const uint8_t* data, uint64_t len);

// Get item count
uint64_t bloom_filter_count(const UniversalBloomFilter* filter);

// Get false positive rate
double bloom_filter_false_positive_rate(const UniversalBloomFilter* filter);

// Reset Bloom Filter
void bloom_filter_reset(UniversalBloomFilter* filter);

#ifdef __cplusplus
}
#endif

#endif // BLOOM_FILTER_H
