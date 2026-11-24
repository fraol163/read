#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

mkdir -p bin

g++ -std=c++17 \
    -Iinclude \
    -Isrc \
    src/buffer_pool.cpp \
    tests/test_buffer_pool.cpp \
    -lgtest -lpthread \
    -o bin/test_runner

./bin/test_runner --gtest_output=xml:test_detail.xml
