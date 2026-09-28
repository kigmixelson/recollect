#!/usr/bin/env bash
# Сборка linux/amd64: тесты, образ, архив с конфигом.
set -euo pipefail
export ARCH="${ARCH:-amd64}"
export PLATFORM="${PLATFORM:-linux/amd64}"
exec "$(cd "$(dirname "$0")" && pwd)/build.sh"
