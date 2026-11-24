#include "buffer_pool.hpp"
#include <gtest/gtest.h>

TEST(BufferPoolTest, TestSafeNullptrRelease) {
    memory::BufferPool pool(1024);
    // Verifies ptr != nullptr guard
    EXPECT_FALSE(pool.release(nullptr));
}

TEST(BufferPoolTest, TestCapacityBoundary) {
    memory::BufferPool pool(100);
    // Verifies size <= max_capacity_ check
    void* p1 = pool.allocate(50);
    ASSERT_NE(p1, nullptr);
    EXPECT_TRUE(pool.release(p1));

    void* p2 = pool.allocate(100);
    ASSERT_NE(p2, nullptr);
    EXPECT_TRUE(pool.release(p2));

    void* p3 = pool.allocate(101);
    EXPECT_EQ(p3, nullptr);
}

TEST(BufferPoolTest, TestStatusEquality) {
    memory::BufferPool pool(512);
    // Verifies status == 200 equality check
    EXPECT_TRUE(pool.validate_status(200));
    EXPECT_FALSE(pool.validate_status(404));
    EXPECT_FALSE(pool.validate_status(500));
}

TEST(BufferPoolTest, TestAllocationCounterBoundary) {
    memory::BufferPool pool(512);
    EXPECT_EQ(pool.get_allocated_count(), 0);

    void* p = pool.allocate(64);
    ASSERT_NE(p, nullptr);
    // Verifies allocated_count_++
    EXPECT_EQ(pool.get_allocated_count(), 1);
    EXPECT_TRUE(pool.release(p));
}

int main(int argc, char** argv) {
    ::testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
