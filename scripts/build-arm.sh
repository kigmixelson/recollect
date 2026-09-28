#!/usr/bin/env bash
# Сборка linux/arm64: тесты, образ, архив с конфигом.
set -euo pipefail
export ARCH="${ARCH:-arm64}"
export PLATFORM="${PLATFORM:-linux/arm64}"
exec "$(cd "$(dirname "$0")" && pwd)/build.sh"
