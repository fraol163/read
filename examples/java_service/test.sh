#!/usr/bin/env bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

mkdir -p bin
javac -d bin \
    src/main/java/com/example/service/OrderValidationService.java \
    src/test/java/com/example/service/OrderValidationServiceTest.java

java -cp bin com.example.service.OrderValidationServiceTest
