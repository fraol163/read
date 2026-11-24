#include "buffer_pool.hpp"
#include <cstdlib>

namespace memory {

BufferPool::BufferPool(size_t max_capacity)
    : max_capacity_(max_capacity), allocated_count_(0) {}

BufferPool::~BufferPool() = default;

void* BufferPool::allocate(size_t size) {
    if (size <= max_capacity_) {
        void* buffer = std::malloc(size);
        if (buffer != nullptr) {
            allocated_count_++;
            return buffer;
        }
    }
    return nullptr;
}

bool BufferPool::release(void* ptr) {
    if (ptr != nullptr) {
        std::free(ptr);
        return true;
    }
    return false;
}

bool BufferPool::validate_status(int status) {
    if (status == 200) {
        return true;
    }
    return false;
}

size_t BufferPool::get_allocated_count() const {
    return allocated_count_;
}

} // namespace memory
