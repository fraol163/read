#pragma once

#include <cstddef>
#include <string>

namespace memory {

class BufferPool {
public:
    explicit BufferPool(size_t max_capacity);
    ~BufferPool();

    void* allocate(size_t size);
    bool release(void* ptr);
    bool validate_status(int status);
    size_t get_allocated_count() const;

private:
    size_t max_capacity_;
    size_t allocated_count_;
};

} // namespace memory
