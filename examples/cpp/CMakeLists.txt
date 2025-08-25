cmake_minimum_required(VERSION 3.20)
project(SecureBufferRust LANGUAGES C)

# Rust settings
set(RUST_DIR "${CMAKE_SOURCE_DIR}/secure/rust")
set(RUST_LIB_DIR "${RUST_DIR}/target/release")
set(RUST_LIB_NAME "securebuffer")

# Always build Rust
add_custom_target(rust_build ALL
    COMMAND cargo build --release
    WORKING_DIRECTORY ${RUST_DIR}
    COMMENT "Building Rust SecureBuffer library..."
)

# Platform-specific lib resolution
if(WIN32)
    set(SECUREBUFFER_LIB "${RUST_LIB_DIR}/${RUST_LIB_NAME}.lib")
elseif(APPLE)
    set(SECUREBUFFER_LIB "${RUST_LIB_DIR}/lib${RUST_LIB_NAME}.a")
else()
    set(SECUREBUFFER_LIB "${RUST_LIB_DIR}/lib${RUST_LIB_NAME}.a")
endif()

# Print info
message(STATUS "Rust lib directory: ${RUST_LIB_DIR}")
message(STATUS "Rust library: ${SECUREBUFFER_LIB}")

# Export for other tools (Go/cgo/FFI)
set(SECUREBUFFER_LIB ${SECUREBUFFER_LIB} CACHE FILEPATH "Path to SecureBuffer Rust library")
