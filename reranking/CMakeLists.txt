cmake_minimum_required(VERSION 3.20)
project(kasumi-reranking
    VERSION 0.1.0
    DESCRIPTION "Kasumi Engine — Latency-constrained LTR reranking service"
    LANGUAGES CXX
)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# ─────────────────────────────────────────────
# Options
# ─────────────────────────────────────────────
option(BUILD_TESTS "Build unit tests" ON)
option(USE_TENSORRT "Enable TensorRT acceleration" OFF)

# ─────────────────────────────────────────────
# Source files
# ─────────────────────────────────────────────
set(RERANKING_SOURCES
    src/main.cpp
)

set(RERANKING_HEADERS
    include/kasumi/reranking/version.h
)

# ─────────────────────────────────────────────
# Main executable
# ─────────────────────────────────────────────
add_executable(kasumi-rerank ${RERANKING_SOURCES})

target_include_directories(kasumi-rerank
    PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

# ─────────────────────────────────────────────
# Tests
# ─────────────────────────────────────────────
if(BUILD_TESTS)
    enable_testing()
    # Tests will be added in Week 4 when the reranking logic is implemented
endif()

# ─────────────────────────────────────────────
# Install
# ─────────────────────────────────────────────
install(TARGETS kasumi-rerank
    RUNTIME DESTINATION bin
)
