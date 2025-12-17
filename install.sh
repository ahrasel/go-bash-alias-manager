#!/usr/bin/env bash
# Simple install script to download the latest release asset and install the binary
set -euo pipefail

REPO_DEFAULT="ahrasel/go-bash-alias-manager"
REPO="${REPO:-$REPO_DEFAULT}"
DEST="${DEST:-/usr/local/bin}"
NAME="bash-alias-manager"

usage() {
    cat <<EOF
Usage: $0 [--repo owner/repo] [--version X.Y.Z] [--dest /path]

Examples:
    curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/raw/main/install.sh | bash
  bash install.sh --version v1.2.3 --dest ~/.local/bin
EOF
}

ASSET_URL=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --repo)
            REPO="$2"; shift 2;;
        --url)
            ASSET_URL="$2"; shift 2;;
        --version)
            VERSION="$2"; shift 2;;
        --dest)
            DEST="$2"; shift 2;;
        --desktop)
            INSTALL_DESKTOP=true; shift;;
        --desktop-dir)
            DESKTOP_DIR="$2"; shift 2;;
        --icons-dir)
            ICONS_DIR="$2"; shift 2;;
        -h|--help)
            usage; exit 0;;
        *)
            echo "Unknown arg: $1"; usage; exit 1;;
    esac
done

VERSION="${VERSION:-}" # empty => latest

if command -v uname >/dev/null 2>&1; then
    UNAME=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
else
    echo "Cannot detect OS/ARCH"; exit 1
fi

case "$UNAME" in
    linux) GOOS=linux;;
    darwin) GOOS=darwin;;
    mingw*|cygwin*|msys*) GOOS=windows;;
    *) echo "Unsupported OS: $UNAME"; exit 1;;
esac

case "$ARCH" in
    x86_64|amd64) GOARCH=amd64;;
    aarch64|arm64) GOARCH=arm64;;
    *) echo "Unsupported ARCH: $ARCH"; exit 1;;
esac

echo "Detected: $GOOS/$GOARCH"

API_URL="https://api.github.com/repos/$REPO/releases"
AUTH_HEADER=""
if [ -n "${GITHUB_TOKEN:-}" ]; then
    AUTH_HEADER="-H 'Authorization: token ${GITHUB_TOKEN}'"
fi

if [ -z "$VERSION" ]; then
    echo "Fetching latest release info..."
    RELEASE_JSON=$(curl -sSf ${AUTH_HEADER} "$API_URL/latest") || { echo "Failed to fetch latest release"; exit 1; }
else
    echo "Fetching release $VERSION info..."
    RELEASE_JSON=$(curl -sSf ${AUTH_HEADER} "$API_URL/tags/$VERSION") || { echo "Failed to fetch release $VERSION"; exit 1; }
fi

ASSET_NAME_PATTERNS=("${NAME}_*_${GOOS}_${GOARCH}.tar.gz" "${NAME}_*_${GOOS}_${GOARCH}.zip")

if [ -z "$ASSET_URL" ]; then
    # Prefer jq if available for robust JSON parsing
    if command -v jq >/dev/null 2>&1; then
        for pat in "${ASSET_NAME_PATTERNS[@]}"; do
            # Convert glob to a regex: replace '*' with '.*'
            regex="$(printf '%s' "$pat" | sed 's/\./\\./g; s/\*/.*/g')"
            ASSET_URL=$(echo "$RELEASE_JSON" | jq -r --arg re "$regex" '.assets[] | select(.name | test($re)) | .browser_download_url' | head -n1 || true)
            if [ -n "$ASSET_URL" ] && [ "$ASSET_URL" != "null" ]; then break; fi
        done
    else
        echo "Note: 'jq' not found — falling back to slower text parsing. For more reliable results install 'jq' (eg. 'sudo apt-get install jq')."
        for pat in "${ASSET_NAME_PATTERNS[@]}"; do
            # Convert glob to regex (escape dots, expand * to .*)
            regex="$(printf '%s' "$pat" | sed 's/\./\\./g; s/\*/.*/g')"
            ASSET_URL=$(echo "$RELEASE_JSON" | grep -Eo '"browser_download_url":\s*"[^"]+"' | cut -d '"' -f4 | grep -E "$regex" || true)
            if [ -n "$ASSET_URL" ]; then break; fi
        done
    fi
fi

# If user provided --url, use that instead
if [ -n "$ASSET_URL" ]; then
    : # found from api
elif [ -n "${ASSET_URL:-}" ]; then
    : # already set
elif [ -n "${ASSET_URL}" ]; then
    :
fi

# If caller provided --url explicitly, it was stored in ASSET_URL earlier; prefer it
if [ -n "$ASSET_URL" ]; then
    :
fi

if [ -z "$ASSET_URL" ]; then
    echo "No release asset found for $GOOS/$GOARCH in repo $REPO"; exit 1
fi

echo "Found asset: $ASSET_URL"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

ARCHIVE="$TMPDIR/asset"
echo "Downloading..."
curl -fsSL "$ASSET_URL" -o "$ARCHIVE"

echo "Extracting..."
mkdir -p "$TMPDIR/extracted"
if echo "$ARCHIVE" | grep -q '\.zip$'; then
    unzip -q "$ARCHIVE" -d "$TMPDIR/extracted"
else
    tar -xzf "$ARCHIVE" -C "$TMPDIR/extracted"
fi

BINPATH=$(find "$TMPDIR/extracted" -type f -name "$NAME*" -perm /111 | head -n1 || true)
if [ -z "$BINPATH" ]; then
    echo "Could not locate binary in archive"; exit 1
fi

DEST_BIN="$DEST/$NAME"
echo "Installing to $DEST_BIN"
mkdir -p "$DEST"

if [ ! -w "$DEST" ]; then
    echo "Destination $DEST is not writable. Trying with sudo..."
    sudo cp "$BINPATH" "$DEST_BIN"
    sudo chmod +x "$DEST_BIN"
else
    cp "$BINPATH" "$DEST_BIN"
    chmod +x "$DEST_BIN"
fi

echo "Installed $NAME to $DEST_BIN"
echo "Run '$DEST_BIN --help' to verify"

if [ "${INSTALL_DESKTOP:-false}" = true ]; then
    # Default per-user locations
    DESKTOP_DIR="${DESKTOP_DIR:-$HOME/.local/share/applications}"
    ICONS_DIR="${ICONS_DIR:-$HOME/.local/share/icons/hicolor/128x128/apps}"

    # Locate desktop template and icon. If the script was piped via stdin, ${BASH_SOURCE[0]} may be unset;
    # in that case prefer raw files from the repository's main branch.
    bs="${BASH_SOURCE[0]:-}"
    RAW_DESKTOP_URL="https://raw.githubusercontent.com/ahrasel/go-bash-alias-manager/main/desktop/bash-alias-manager.desktop"
    RAW_ICON_URL="https://raw.githubusercontent.com/ahrasel/go-bash-alias-manager/main/assets/icon.svg"

    if [ -n "$bs" ]; then
        SCRIPT_DIR="$(cd "$(dirname "$bs")" && pwd)"
        local_desktop="$SCRIPT_DIR/desktop/bash-alias-manager.desktop"
        local_icon="$SCRIPT_DIR/assets/icon.svg"
        if [ -f "$local_desktop" ]; then
            FINAL_DESKTOP="$local_desktop"
        else
            FINAL_DESKTOP="$RAW_DESKTOP_URL"
        fi
        if [ -f "$local_icon" ]; then
            FINAL_ICON="$local_icon"
        else
            FINAL_ICON="$RAW_ICON_URL"
        fi
    else
        # Running from stdin; use raw URLs
        FINAL_DESKTOP="$RAW_DESKTOP_URL"
        FINAL_ICON="$RAW_ICON_URL"
    fi

    echo "Installing desktop entry to $DESKTOP_DIR and icon to $ICONS_DIR"
    mkdir -p "$DESKTOP_DIR" "$ICONS_DIR"

    # Prepare desktop file with full Exec path
    DESKTOP_TARGET="$DESKTOP_DIR/bash-alias-manager.desktop"
    # If DESKTOP_SRC is a URL, download it to a temp file first
    if printf '%s' "$FINAL_DESKTOP" | grep -qE '^https?://'; then
        tmpd=$(mktemp -d)
        trap 'rm -rf "$tmpd"' EXIT
        curl -fsSL "$FINAL_DESKTOP" -o "$tmpd/bash-alias-manager.desktop"
        awk -v execpath="$DEST_BIN" 'BEGIN{FS=OFS="="} /^Exec=/{$2=execpath} {print}' "$tmpd/bash-alias-manager.desktop" > "$DESKTOP_TARGET"
        rm -rf "$tmpd"
    else
        awk -v execpath="$DEST_BIN" 'BEGIN{FS=OFS="="} /^Exec=/{$2=execpath} {print}' "$FINAL_DESKTOP" > "$DESKTOP_TARGET"
    fi
    # Ensure Icon line exists and points to our icon name
    if ! grep -q "^Icon=" "$DESKTOP_TARGET"; then
        echo "Icon=bash-alias-manager" >> "$DESKTOP_TARGET"
    else
        sed -i "s/^Icon=.*/Icon=bash-alias-manager/" "$DESKTOP_TARGET"
    fi

    # Copy icon (svg preferred) to icons dir
    ICON_TARGET="$ICONS_DIR/bash-alias-manager.svg"
    if printf '%s' "$FINAL_ICON" | grep -qE '^https?://'; then
        curl -fsSL "$FINAL_ICON" -o "$ICON_TARGET"
    else
        cp "$FINAL_ICON" "$ICON_TARGET"
    fi

    # If ImageMagick's convert is available, produce a PNG as well
    if command -v convert >/dev/null 2>&1; then
        PNG_TARGET="$ICONS_DIR/bash-alias-manager.png"
        convert -background none "$ICON_SRC" -resize 128x128 "$PNG_TARGET" || true
    fi

    # Update caches if available
    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database "$DESKTOP_DIR" || true
    fi
    if command -v gtk-update-icon-cache >/dev/null 2>&1; then
        gtk-update-icon-cache -f -t "$(dirname "$ICONS_DIR")" || true
    fi

    echo "Desktop menu entry installed: $DESKTOP_TARGET"
fi

exit 0
