#!/usr/bin/env bash

# Build release artifacts for GitHub Releases
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

NAME="bash-alias-manager"
DIST_DIR="$ROOT_DIR/dist"
SRC_DIR="$ROOT_DIR"

# Detect version from git tag, fallback to commit short
VERSION="$(git describe --tags --always --dirty 2>/dev/null || true)"
if [ -z "$VERSION" ]; then
    VERSION="dev"
fi

echo "Building release artifacts for $NAME version $VERSION"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

TARGETS=("linux/amd64")

for t in "${TARGETS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "$t"
    echo "- Building for $GOOS/$GOARCH"

    OUTDIR="$DIST_DIR/${NAME}_${VERSION}_${GOOS}_${GOARCH}"
    mkdir -p "$OUTDIR"

    BINNAME="$NAME"
    if [ "$GOOS" = "windows" ]; then
        BINNAME="${NAME}.exe"
    fi

    # If a prebuilt binary exists in 'prime/bin', prefer that (snap build output)
    if [ -x "${ROOT_DIR}/prime/bin/$NAME" ] && [ "$GOOS" = "linux" ] && [ "$GOARCH" = "amd64" ]; then
        echo "- Using prebuilt binary from prime/bin"
        cp "${ROOT_DIR}/prime/bin/$NAME" "$OUTDIR/$BINNAME"
    else
        # Try building; fynes may require CGO and platform-specific deps, so this may fail
        env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-s -w" -o "$OUTDIR/$BINNAME" ./... || true
    fi

    # Copy docs and installer script
    cp README.md "$OUTDIR/"
    cp LICENSE* "$OUTDIR/" 2>/dev/null || true
    if [ -f install.sh ]; then
        cp install.sh "$OUTDIR/"
    fi

    # Archive
    pushd "$DIST_DIR" >/dev/null
    if [ "$GOOS" = "windows" ]; then
        zip -r "${NAME}_${VERSION}_${GOOS}_${GOARCH}.zip" "${NAME}_${VERSION}_${GOOS}_${GOARCH}" >/dev/null
    else
        tar -czf "${NAME}_${VERSION}_${GOOS}_${GOARCH}.tar.gz" "${NAME}_${VERSION}_${GOOS}_${GOARCH}" >/dev/null
    fi
    popd >/dev/null

done

echo "Generating checksums"
pushd "$DIST_DIR" >/dev/null
ARTIFACTS=()
for f in *.{tar.gz,zip}; do
    [ -e "$f" ] || continue
    ARTIFACTS+=("$f")
done
if [ ${#ARTIFACTS[@]} -gt 0 ]; then
    shasum -a 256 "${ARTIFACTS[@]}" | tee "${NAME}_${VERSION}_SHA256SUMS"
else
    echo "No artifacts found to checksum"
fi
popd >/dev/null

echo "Artifacts are in: $DIST_DIR"

exit 0
