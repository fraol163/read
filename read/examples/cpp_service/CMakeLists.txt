cmake_minimum_required(VERSION 3.14)
project(cpp_service CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include(CTest)
enable_testing()

find_package(GTest REQUIRED)

add_library(buffer_pool src/buffer_pool.cpp)
target_include_directories(buffer_pool PUBLIC include src)

add_executable(test_buffer_pool tests/test_buffer_pool.cpp)
target_link_libraries(test_buffer_pool PRIVATE buffer_pool GTest::GTest GTest::Main pthread)

add_test(NAME BufferPoolTests COMMAND test_buffer_pool)
