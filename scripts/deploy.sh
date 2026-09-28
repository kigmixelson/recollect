#!/usr/bin/env bash
# Распаковка архива сборки, загрузка образа и запуск на целевой машине.
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/recollect}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/healthz}"
HEALTH_TRIES="${HEALTH_TRIES:-20}"

usage() {
  cat <<'EOF'
использование:
  deploy.sh [архив.tar.gz]

без аргумента скрипт ищет image.tar.gz в текущем каталоге
  (так работает копия deploy.sh внутри архива сборки).

переменные:
  INSTALL_DIR   каталог установки, по умолчанию /opt/recollect
  HEALTH_URL    проверка после запуска
EOF
}

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "нужна команда: $1" >&2
    exit 1
  fi
}

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    echo "нужен Docker Compose (docker compose или docker-compose)" >&2
    exit 1
  fi
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

need docker
need tar

if ! docker info >/dev/null 2>&1; then
  echo "docker не запущен или нет доступа к демону" >&2
  exit 1
fi

SRC=""
TMP=""
cleanup() {
  if [[ -n "${TMP}" && -d "${TMP}" ]]; then
    rm -rf "${TMP}"
  fi
}
trap cleanup EXIT

if [[ $# -ge 1 ]]; then
  ARCHIVE="$1"
  if [[ ! -f "${ARCHIVE}" ]]; then
    echo "нет файла: ${ARCHIVE}" >&2
    exit 1
  fi
  TMP="$(mktemp -d)"
  tar -xzf "${ARCHIVE}" -C "${TMP}"
  SRC="$(find "${TMP}" -name image.tar.gz | head -n 1)"
  if [[ -z "${SRC}" ]]; then
    echo "в архиве нет image.tar.gz" >&2
    exit 1
  fi
  SRC="$(dirname "${SRC}")"
elif [[ -f ./image.tar.gz && -f ./docker-compose.yml ]]; then
  SRC="$(pwd)"
else
  usage >&2
  exit 1
fi

echo "==> установка в ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"

echo "==> загрузка образа"
gzip -dc "${SRC}/image.tar.gz" | docker load

cp "${SRC}/docker-compose.yml" "${INSTALL_DIR}/docker-compose.yml"
if [[ -f "${SRC}/VERSION" ]]; then
  cp "${SRC}/VERSION" "${INSTALL_DIR}/VERSION"
fi

NEW_ENV="${SRC}/.env"
if [[ -f "${INSTALL_DIR}/.env" ]]; then
  echo "==> сохраняю существующий ${INSTALL_DIR}/.env, обновляю RECOLLECT_IMAGE"
  if [[ -f "${NEW_ENV}" ]] && grep -q '^RECOLLECT_IMAGE=' "${NEW_ENV}"; then
    IMAGE_LINE="$(grep '^RECOLLECT_IMAGE=' "${NEW_ENV}" | tail -n 1)"
    if grep -q '^RECOLLECT_IMAGE=' "${INSTALL_DIR}/.env"; then
      tmp_env="$(mktemp)"
      sed "s|^RECOLLECT_IMAGE=.*|${IMAGE_LINE}|" "${INSTALL_DIR}/.env" > "${tmp_env}"
      mv "${tmp_env}" "${INSTALL_DIR}/.env"
    else
      printf '\n%s\n' "${IMAGE_LINE}" >> "${INSTALL_DIR}/.env"
    fi
  fi
else
  cp "${NEW_ENV}" "${INSTALL_DIR}/.env"
fi

echo "==> запуск"
(
  cd "${INSTALL_DIR}"
  compose up -d
)

echo "==> проверка ${HEALTH_URL}"
ok=0
for _ in $(seq 1 "${HEALTH_TRIES}"); do
  if command -v curl >/dev/null 2>&1; then
    if curl -fsS "${HEALTH_URL}" >/dev/null 2>&1; then
      ok=1
      break
    fi
  elif command -v wget >/dev/null 2>&1; then
    if wget -qO- "${HEALTH_URL}" >/dev/null 2>&1; then
      ok=1
      break
    fi
  else
    echo "нет curl/wget, проверяйте healthz вручную"
    ok=1
    break
  fi
  sleep 1
done

if [[ "${ok}" -ne 1 ]]; then
  echo "сервис не ответил на ${HEALTH_URL}" >&2
  (
    cd "${INSTALL_DIR}"
    compose logs --tail 80 recollect || true
  )
  exit 1
fi

echo "готово: ${INSTALL_DIR}"
echo "логи: cd ${INSTALL_DIR} && docker compose logs -f recollect"
