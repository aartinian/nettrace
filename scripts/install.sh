#!/usr/bin/env bash
set -euo pipefail

REPO="${NETTRACE_REPO:-aartinian/nettrace}"
VERSION="${1:-latest}"

resolve_version() {
  if [[ "$VERSION" != "latest" ]]; then
    echo "$VERSION"
    return
  fi

  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' \
    | head -n1
}

map_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "unsupported operating system" >&2
      exit 1
      ;;
  esac
}

map_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture" >&2
      exit 1
      ;;
  esac
}

VERSION_TAG="$(resolve_version)"
if [[ -z "$VERSION_TAG" ]]; then
  echo "unable to resolve release version" >&2
  exit 1
fi

OS="$(map_os)"
ARCH="$(map_arch)"

ARCHIVE="nettrace_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION_TAG}/${ARCHIVE}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fL "$URL" -o "$TMP_DIR/nettrace.tar.gz"
tar -xzf "$TMP_DIR/nettrace.tar.gz" -C "$TMP_DIR"

install "$TMP_DIR/nettrace" /usr/local/bin/nettrace

echo "installed nettrace ${VERSION_TAG} to /usr/local/bin/nettrace"
