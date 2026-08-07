#!/bin/sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "usage: $0 VERSION [OUTPUT_DIRECTORY]" >&2
    exit 2
fi

VERSION=${1#v}
OUTPUT_DIRECTORY=${2:-dist}
ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"
case "$OUTPUT_DIRECTORY" in
    /*) ;;
    *) OUTPUT_DIRECTORY="$ROOT/$OUTPUT_DIRECTORY" ;;
esac
case "$OUTPUT_DIRECTORY" in
    /|"$ROOT") echo "refusing unsafe output directory: $OUTPUT_DIRECTORY" >&2; exit 2 ;;
esac

rm -rf "$OUTPUT_DIRECTORY"
mkdir -p "$OUTPUT_DIRECTORY"
CENTER="$OUTPUT_DIRECTORY/relayward-plugin-xray-center-linux-amd64"
NODE="$OUTPUT_DIRECTORY/relayward-plugin-xray-node-linux-amd64"
UI="$OUTPUT_DIRECTORY/relayward-plugin-xray-ui.tar.gz"
npm --prefix "$ROOT/ui" run build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath -buildvcs=false \
    -ldflags "-s -w -buildid= -X main.version=$VERSION" \
    -o "$CENTER" ./cmd/relayward-plugin-xray
if [ "$("$CENTER" version)" != "$VERSION" ]; then
    echo "plugin binary reports an unexpected version" >&2
    exit 1
fi
cp "$CENTER" "$NODE"
chmod 0755 "$CENTER" "$NODE"
tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner \
    -cf - -C "$ROOT/ui/dist" . | gzip -n > "$UI"
go run ./cmd/relayward-plugin-xray-release -dist "$OUTPUT_DIRECTORY" -version "$VERSION"
(
    cd "$OUTPUT_DIRECTORY"
    sha256sum \
        relayward-plugin-xray-center-linux-amd64 \
        relayward-plugin-xray-node-linux-amd64 \
        relayward-plugin-xray-ui.tar.gz \
        relayward-plugin.json > SHA256SUMS
)
