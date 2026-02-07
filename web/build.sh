#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
FRONTEND_DIR="${ROOT_DIR}/web/vtuber"
DIST_DIR="${FRONTEND_DIR}/dist/web"
TARGET_DIR="${ROOT_DIR}/webassets/vtuber"

if [[ ! -d "${FRONTEND_DIR}" ]]; then
  echo "frontend dir not found: ${FRONTEND_DIR}" >&2
  exit 1
fi

if [[ ! -d "${FRONTEND_DIR}/node_modules" ]]; then
  echo "node_modules not found, running npm install..."
  (cd "${FRONTEND_DIR}" && npm install)
fi

echo "building frontend..."
(cd "${FRONTEND_DIR}" && npm run build:web)

if [[ ! -d "${DIST_DIR}" ]]; then
  echo "dist not found: ${DIST_DIR}" >&2
  exit 1
fi

echo "syncing build to ${TARGET_DIR}"
mkdir -p "${TARGET_DIR}"
rm -rf "${TARGET_DIR}"/*
cp -a "${DIST_DIR}/." "${TARGET_DIR}/"

echo "done"
