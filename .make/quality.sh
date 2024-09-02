#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
	echo "$unformatted"
	exit 1
fi

go vet ./...
./.make/test.sh
./.make/build.sh
