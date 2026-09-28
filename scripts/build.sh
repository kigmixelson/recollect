#!/usr/bin/env bash
# Общая сборка: тесты, кросс-компиляция образа, архив с образом и конфигом.
# Вызывается из build-arm.sh / build-amd.sh либо с ARCH и PLATFORM.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ARCH="${ARCH:?задайте ARCH (arm64 или amd64)}"
PLATFORM="${PLATFORM:-linux/${ARCH}}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || date +%Y%m%d%H%M)}"
IMAGE="recollect:${VERSION}-${ARCH}"
DIST="${ROOT}/dist"
BUNDLE="recollect-${VERSION}-linux-${ARCH}"
STAGE="${DIST}/${BUNDLE}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "нужна команда: $1" >&2
    exit 1
  fi
}

need docker
need tar
need gzip

if ! docker info >/dev/null 2>&1; then
  echo "docker не запущен или нет доступа к демону" >&2
  exit 1
fi

echo "==> версия ${VERSION}, платформа ${PLATFORM}"

echo "==> тесты"
if command -v go >/dev/null 2>&1; then
  go test ./...
else
  docker run --rm \
    -v "${ROOT}:/src" \
    -w /src \
    golang:1.23-alpine \
    go test ./...
fi

echo "==> сборка образа ${IMAGE} (кросс-компиляция в ${PLATFORM})"
export DOCKER_BUILDKIT=1
if docker buildx version >/dev/null 2>&1; then
  docker buildx build \
    --platform "${PLATFORM}" \
    --build-arg TARGETOS=linux \
    --build-arg TARGETARCH="${ARCH}" \
    --load \
    -t "${IMAGE}" \
    -t "recollect:local-${ARCH}" \
    .
else
  docker build \
    --platform "${PLATFORM}" \
    --build-arg TARGETOS=linux \
    --build-arg TARGETARCH="${ARCH}" \
    -t "${IMAGE}" \
    -t "recollect:local-${ARCH}" \
    .
fi

echo "==> упаковка ${BUNDLE}"
rm -rf "${STAGE}"
mkdir -p "${STAGE}"

docker save "${IMAGE}" | gzip > "${STAGE}/image.tar.gz"
cp "${ROOT}/docker-compose.yml" "${STAGE}/docker-compose.yml"
cp "${ROOT}/scripts/deploy.sh" "${STAGE}/deploy.sh"
cp "${ROOT}/nginx/recollect.conf" "${STAGE}/recollect.conf"
chmod +x "${STAGE}/deploy.sh"

cat > "${STAGE}/.env" <<EOF
RECOLLECT_IMAGE=${IMAGE}
LISTEN_ADDR=:8080
HTTP_TIMEOUT=30s
SAYMON_CONCURRENCY=8
SAYMON_TLS_INSECURE=false
EOF

cat > "${STAGE}/VERSION" <<EOF
version=${VERSION}
image=${IMAGE}
platform=${PLATFORM}
arch=${ARCH}
EOF

mkdir -p "${DIST}"
ARCHIVE="${DIST}/${BUNDLE}.tar.gz"
tar -C "${DIST}" -czf "${ARCHIVE}" "${BUNDLE}"

echo
echo "архив: ${ARCHIVE}"
echo "скопируйте его на целевую машину и запустите:"
echo "  ./scripts/deploy.sh ${ARCHIVE}"
echo "или распакуйте и выполните ./deploy.sh внутри каталога"
