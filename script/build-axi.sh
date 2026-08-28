#!/usr/bin/env bash
# Build the gh-axi Go binary (fork of cli/cli with TOON output).
#
# Usage:
#   script/build-axi.sh                # builds ./gh-axi with 0.1.0-<commit>
#   VERSION=0.2.0 script/build-axi.sh  # custom version
#   OUTPUT=/tmp/gh-axi script/build-axi.sh
#   script/build-axi.sh --install      # installs to ~/.local/bin/gh-axi
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-0.1.0}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"
FULL_VERSION="${VERSION}-${COMMIT}"
DATE="$(date +%Y-%m-%d)"

if [[ "${1:-}" == "--install" ]]; then
  OUTPUT="${HOME}/.local/bin/gh-axi"
else
  OUTPUT="${OUTPUT:-gh-axi}"
fi

# gh's auto-update check is disabled by default: a plain `go build` omits the
# `updateable` build tag (Homebrew's formula is what adds it), so no update
# messages appear. If a future upstream merge re-enables it, verify the tag.
echo "building gh-axi ${FULL_VERSION} (${DATE}) -> ${OUTPUT}"
go build -o "${OUTPUT}" \
  -ldflags "-X github.com/cli/cli/v2/internal/build.Version=${FULL_VERSION} -X github.com/cli/cli/v2/internal/build.Date=${DATE}" \
  ./cmd/gh
echo "ok: ${OUTPUT}"
