#!/bin/sh
# 安装 /app/anchor：RELEASE_VERSION=local 用 build context bin/，否则从 GitHub Release 下载。
set -eu

RELEASE_VERSION="${1:-latest}"
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)  TARGETARCH=amd64 ;;
  aarch64) TARGETARCH=arm64 ;;
  *) echo "unsupported architecture: ${ARCH}" >&2; exit 1 ;;
esac

if [ "${RELEASE_VERSION}" = "local" ]; then
  SRC="/tmp/anchor-bin/anchor-linux-${TARGETARCH}"
  if [ ! -f "${SRC}" ]; then
    echo "RELEASE_VERSION=local requires ${SRC} (run: make build-linux)" >&2
    exit 1
  fi
  echo "Using local binary anchor-linux-${TARGETARCH}"
  cp "${SRC}" /app/anchor
elif [ "${RELEASE_VERSION}" = "latest" ]; then
  URL="https://github.com/P0m32Kun/Anchor/releases/latest/download/anchor-linux-${TARGETARCH}"
  echo "Downloading anchor ${RELEASE_VERSION} for ${TARGETARCH}..."
  curl -fsSL -o /app/anchor "${URL}"
else
  URL="https://github.com/P0m32Kun/Anchor/releases/download/${RELEASE_VERSION}/anchor-linux-${TARGETARCH}"
  echo "Downloading anchor ${RELEASE_VERSION} for ${TARGETARCH}..."
  curl -fsSL -o /app/anchor "${URL}"
fi

chmod +x /app/anchor
