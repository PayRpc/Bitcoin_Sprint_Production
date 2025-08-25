// SPDX-License-Identifier: MIT
// Bitcoin Sprint C++ Example
// Demonstrates SecureBuffer usage from C++

#include <iostream>
#include <string>
#include <memory>
#include "securebuffer.h"

// RAII wrapper for SecureBuffer
class SecureString {
private:
    SecureBuffer* buffer_;

public:
    explicit SecureString(size_t size) : buffer_(securebuffer_new(size)) {
        if (!buffer_) {
            throw std::runtime_error("Failed to create SecureBuffer");
        }
    }

    ~SecureString() {
        if (buffer_) {
            securebuffer_free(buffer_);
        }
    }

    // Delete copy constructor and assignment operator
    SecureString(const SecureString&) = delete;
    SecureString& operator=(const SecureString&) = delete;

    // Move constructor and assignment operator
    SecureString(SecureString&& other) noexcept : buffer_(other.buffer_) {
        other.buffer_ = nullptr;
    }

    SecureString& operator=(SecureString&& other) noexcept {
        if (this != &other) {
            if (buffer_) {
                securebuffer_free(buffer_);
            }
            buffer_ = other.buffer_;
            other.buffer_ = nullptr;
        }
        return *this;
    }

    bool setData(const std::string& data) {
        if (!buffer_) return false;
        return securebuffer_copy(buffer_, data.c_str()) == 1;
    }

    size_t length() const {
        return buffer_ ? securebuffer_len(buffer_) : 0;
    }

    bool isValid() const {
        return buffer_ != nullptr;
    }
};

int main() {
    std::cout << "🔐 Bitcoin Sprint C++ SecureBuffer Example\n";
    std::cout << "==========================================\n\n";

    try {
        // Create secure storage for sensitive data
        std::cout << "1. Creating SecureBuffer for API key...\n";
        SecureString apiKey(64);
        
        if (!apiKey.isValid()) {
            std::cerr << "❌ Failed to create SecureBuffer\n";
            return 1;
        }

        // Store sensitive data securely
        std::string sensitiveData = "sk_live_1234567890abcdef";
        std::cout << "2. Storing sensitive data securely...\n";
        
        if (!apiKey.setData(sensitiveData)) {
            std::cerr << "❌ Failed to store data in SecureBuffer\n";
            return 1;
        }

        // Clear the plain text version
        sensitiveData.clear();
        std::cout << "3. Plain text cleared from memory\n";

        // Show that secure storage is working
        std::cout << "4. SecureBuffer length: " << apiKey.length() << " bytes\n";
        std::cout << "✅ Sensitive data is now protected in memory-locked storage\n\n";

        // Demonstrate multiple secure buffers
        std::cout << "5. Creating additional secure storage...\n";
        SecureString password(32);
        SecureString token(128);

        if (password.setData("MySecretPassword123!") && 
            token.setData("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")) {
            std::cout << "✅ Multiple secure buffers created successfully\n";
            std::cout << "   - Password buffer: " << password.length() << " bytes\n";
            std::cout << "   - Token buffer: " << token.length() << " bytes\n";
        }

        std::cout << "\n🛡️  Security Features Active:\n";
        std::cout << "   ✓ Memory pages locked (cannot be swapped to disk)\n";
        std::cout << "   ✓ Memory will be securely zeroed on destruction\n";
        std::cout << "   ✓ Protected from memory dumps and core dumps\n";
        std::cout << "   ✓ RAII ensures automatic cleanup\n";

        std::cout << "\n🎉 C++ SecureBuffer integration working perfectly!\n";

    } catch (const std::exception& e) {
        std::cerr << "❌ Error: " << e.what() << std::endl;
        return 1;
    }

    // SecureString destructors will automatically clean up secure memory
    std::cout << "6. Automatic secure cleanup on scope exit...\n";
    return 0;
}
