#!/bin/sh
set -e

ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

VERSION=$(curl -fsSL https://api.github.com/repos/muuvmuuv/php-tuner/releases/latest | grep '"tag_name"' | cut -d'"' -f4 | tr -d 'v')
URL="https://github.com/muuvmuuv/php-tuner/releases/latest/download/php-tuner-${VERSION}-linux-${ARCH}.tar.gz"

curl -fsSL "$URL" | tar xz
./php-tuner help
