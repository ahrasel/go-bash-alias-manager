#!/bin/bash

# Bash Alias Manager Snap Release Script
# Automates building, uploading, and releasing the snap

set -e  # Exit on any error

# Configuration
SNAP_NAME="bash-alias-manager"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SNAP_FILE="${SNAP_NAME}_*.snap"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  build          Build the snap locally"
    echo "  upload         Upload snap to store (requires login)"
    echo "  release        Release snap to channel (requires login)"
    echo "  status         Check snap status in store"
    echo "  clean          Clean build artifacts"
    echo "  full           Build, upload, and release (requires login)"
    echo "  package        Build GitHub release artifacts (tar.gz/zip + checksums)"
    echo "  gh-release     Create a GitHub release and upload assets (requires GH CLI or GITHUB_TOKEN)"
    echo "  gh-full        Package and publish GitHub release (build -> create release -> upload assets)"
    echo "  help           Show this help"
    echo ""
    echo "Options:"
    echo "  --channel CH   Release channel (default: stable)"
    echo "  --version VER  Version to release"
    echo "  --yes          Skip confirmation prompts"
    echo ""
    echo "Examples:"
    echo "  $0 build"
    echo "  $0 upload"
    echo "  $0 release --channel beta"
    echo "  $0 full --channel stable"
}

# Build GitHub release artifacts
package_release() {
    log_info "Packaging release artifacts"
    scripts/build_release.sh
    log_success "Packaging complete"
}

# Create GitHub release using gh (preferred) or GitHub API
gh_create_release() {
    VERSION_TAG="$1"
    DIST_DIR="$PROJECT_DIR/dist"

    if [ ! -d "$DIST_DIR" ] || [ -z "$(ls -A $DIST_DIR 2>/dev/null)" ]; then
        log_error "No artifacts found in $DIST_DIR. Run './release.sh package' first or use 'gh-full' to build and publish in one step."
        exit 1
    fi

    if command -v gh >/dev/null 2>&1; then
        log_info "Creating GitHub release (gh) $VERSION_TAG"
        gh release create "$VERSION_TAG" "$DIST_DIR"/*.{tar.gz,zip} --title "$VERSION_TAG" --notes "Release $VERSION_TAG" || true
        log_success "GitHub release created via gh"
        return 0
    fi

    # Fallback: use GitHub API if GITHUB_TOKEN is present
    if [ -z "${GITHUB_TOKEN:-}" ]; then
        log_error "gh CLI not found and GITHUB_TOKEN not set; cannot create GitHub release"
        exit 1
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log_error "The 'jq' command is required for GitHub API fallback. Install it or use the 'gh' CLI."
        exit 1
    fi

    log_info "Creating GitHub release via API: $VERSION_TAG"
    # create release
    owner_repo="${GITHUB_REPO:-ahrasel/go-bash-alias-manager}"
    api_url="https://api.github.com/repos/$owner_repo/releases"
    post_data=$(jq -n --arg tag "$VERSION_TAG" --arg name "$VERSION_TAG" --arg body "Release $VERSION_TAG" '{tag_name:$tag, name:$name, body:$body, draft:false, prerelease:false}')
    release_response=$(curl -sS -H "Authorization: token $GITHUB_TOKEN" -H "Content-Type: application/json" -d "$post_data" "$api_url")
    upload_url=$(echo "$release_response" | jq -r .upload_url | sed -e 's/{?name,label}//')
    if [ -z "$upload_url" ] || [ "$upload_url" = "null" ]; then
        log_error "Failed to create release via API"
        echo "$release_response" | jq .
        exit 1
    fi

    # upload assets
    for f in "$DIST_DIR"/*.{tar.gz,zip}; do
        [ -e "$f" ] || continue
        fname=$(basename "$f")
        log_info "Uploading $fname"
        curl -sS -H "Authorization: token $GITHUB_TOKEN" -H "Content-Type: application/octet-stream" --data-binary "@$f" "$upload_url?name=$fname"
    done

    log_success "GitHub release created and assets uploaded"
}

# Function to check if snapcraft is logged in
check_login() {
    if ! snapcraft whoami >/dev/null 2>&1; then
        log_error "Not logged in to snapcraft. Please run: snapcraft login"
        exit 1
    fi
}

# Function to build snap
build_snap() {
    local quiet="${1:-false}"

    if [ "$quiet" = false ]; then
        log_info "Building snap..."
    fi

    cd "$PROJECT_DIR"

    # Clean previous builds
    rm -f ${SNAP_FILE}

    # Build the snap
    if snapcraft pack --destructive-mode; then
        SNAP_FILE_PATH=$(ls -t ${SNAP_FILE} 2>/dev/null | head -1)
        if [ -n "$SNAP_FILE_PATH" ]; then
            if [ "$quiet" = false ]; then
                log_success "Snap built successfully: $SNAP_FILE_PATH"
            fi
            echo "$SNAP_FILE_PATH"
        else
            log_error "Snap file not found after build"
            exit 1
        fi
    else
        log_error "Snap build failed"
        exit 1
    fi
}

# Function to upload snap
upload_snap() {
    local snap_file="$1"

    if [ -z "$snap_file" ]; then
        log_error "No snap file provided for upload"
        exit 1
    fi

    check_login

    log_info "Uploading snap: $snap_file"
    if snapcraft upload "$snap_file"; then
        log_success "Snap uploaded successfully"
    else
        log_error "Snap upload failed"
        exit 1
    fi
}

# Function to release snap
release_snap() {
    local channel="$1"
    local version="$2"

    check_login

    log_info "Releasing snap to $channel channel"

    if [ -n "$version" ]; then
        if snapcraft release "$SNAP_NAME" "$version" "$channel"; then
            log_success "Snap released to $channel channel (version: $version)"
        else
            log_error "Snap release failed"
            exit 1
        fi
    else
        # Get the latest revision
        local revision
        revision=$(snapcraft status "$SNAP_NAME" | grep -E "^[0-9]+" | head -1 | awk '{print $1}')
        if [ -z "$revision" ]; then
            log_error "Could not determine latest revision"
            exit 1
        fi

        log_info "Using latest revision: $revision"
        if snapcraft release "$SNAP_NAME" "$revision" "$channel"; then
            log_success "Snap released to $channel channel (revision: $revision)"
        else
            log_error "Snap release failed"
            exit 1
        fi
    fi
}

# Function to check status
check_status() {
    log_info "Checking snap status..."
    snapcraft status "$SNAP_NAME"
}

# Function to clean build artifacts
clean_build() {
    log_info "Cleaning build artifacts..."
    cd "$PROJECT_DIR"
    rm -f ${SNAP_FILE}
    rm -rf build/
    rm -rf parts/
    rm -rf prime/
    rm -rf stage/
    log_success "Build artifacts cleaned"
}

# Main script logic
main() {
    local command=""
    local channel="stable"
    local version=""
    local skip_confirm=false
    local snap_file=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            build|upload|release|status|clean|full|help)
                command="$1"
                shift
                ;;
            --channel)
                channel="$2"
                shift 2
                ;;
            --version)
                version="$2"
                shift 2
                ;;
            --yes)
                skip_confirm=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Show usage if no command provided
    if [ -z "$command" ]; then
        show_usage
        exit 1
    fi

    # Handle help command
    if [ "$command" = "help" ]; then
        show_usage
        exit 0
    fi

    # Handle different commands
    case "$command" in
        build)
            snap_file=$(build_snap)
            echo "Built snap: $snap_file"
            ;;
        package)
            package_release
            ;;
        gh-release)
            # Determine tag to use
            if [ -n "$version" ]; then
                tag="$version"
            else
                tag="$(git describe --tags --always)"
            fi
            gh_create_release "$tag"
            ;;
        gh-full)
            # Build artifacts and publish to GitHub
            package_release
            if [ -n "$version" ]; then
                tag="$version"
            else
                tag="$(git describe --tags --always)"
            fi
            gh_create_release "$tag"
            ;;
        upload)
            if [ -z "$snap_file" ]; then
                snap_file=$(ls -t ${SNAP_FILE} 2>/dev/null | head -1)
                if [ -z "$snap_file" ]; then
                    log_error "No snap file found. Run 'build' first."
                    exit 1
                fi
            fi
            upload_snap "$snap_file"
            ;;
        release)
            release_snap "$channel" "$version"
            ;;
        status)
            check_status
            ;;
        clean)
            clean_build
            ;;
        full)
            log_info "Starting full release process (build -> upload -> release)"

            if [ "$skip_confirm" = false ]; then
                read -p "This will build, upload, and release the snap. Continue? (y/N): " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    log_info "Aborted by user"
                    exit 0
                fi
            fi

            # Build (quiet mode to avoid capturing colored output)
            snap_file=$(build_snap true)

            # Upload
            upload_snap "$snap_file"

            # Release
            release_snap "$channel" "$version"

            log_success "Full release process completed!"
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"