cmake_minimum_required(VERSION 3.20)
project(BitcoinSprintCpp VERSION 1.0.0 LANGUAGES CXX)

# C++ Standard
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Build configuration
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

# Platform detection
if(WIN32)
    set(LIB_EXT "dll")
    set(LIB_PREFIX "")
else()
    set(LIB_EXT "so")
    set(LIB_PREFIX "lib")
endif()

# Rust library paths
set(RUST_DIR "${CMAKE_SOURCE_DIR}/secure/rust")
set(RUST_INCLUDE_DIR "${RUST_DIR}/include")
set(RUST_LIB_DIR "${RUST_DIR}/target/release")
set(RUST_LIB_NAME "securebuffer")

# Find Rust library
find_library(SECUREBUFFER_LIB
    NAMES ${RUST_LIB_NAME}
    PATHS ${RUST_LIB_DIR}
    NO_DEFAULT_PATH
    REQUIRED
)

# Include directories
include_directories(${RUST_INCLUDE_DIR})

# C++ Example executable
add_executable(cpp-example
    examples/cpp/main.cpp
)

# Link libraries
target_link_libraries(cpp-example ${SECUREBUFFER_LIB})

# Compiler-specific options
if(MSVC)
    target_compile_options(cpp-example PRIVATE /W4)
else()
    target_compile_options(cpp-example PRIVATE -Wall -Wextra -pedantic)
endif()

# Custom target to build Rust library first
add_custom_target(rust_build
    COMMAND cargo build --release
    WORKING_DIRECTORY ${RUST_DIR}
    COMMENT "Building Rust SecureBuffer library..."
)

# Make C++ example depend on Rust build
add_dependencies(cpp-example rust_build)

# Installation
install(TARGETS cpp-example
    RUNTIME DESTINATION bin
)

# Custom targets for convenience
add_custom_target(demo
    COMMAND ${CMAKE_COMMAND} --build . --target cpp-example
    COMMAND $<TARGET_FILE:cpp-example>
    DEPENDS cpp-example
    COMMENT "Building and running C++ example..."
)

# Print build information
message(STATUS "Bitcoin Sprint C++ Build Configuration")
message(STATUS "=====================================")
message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "C++ standard: ${CMAKE_CXX_STANDARD}")
message(STATUS "Rust directory: ${RUST_DIR}")
message(STATUS "Rust library: ${SECUREBUFFER_LIB}")
message(STATUS "Platform: ${CMAKE_SYSTEM_NAME}")
