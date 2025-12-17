#!/bin/bash

# Bash Alias Manager Release Script
# Automates building release artifacts and publishing to GitHub Releases

set -e  # Exit on any error

# Configuration
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
    echo "  package        Build release artifacts (tar.gz + checksums)"
    echo "  gh-release     Create a GitHub release and upload assets (requires GH CLI or GITHUB_TOKEN)"
    echo "  gh-full        Package and publish GitHub release (build -> create release -> upload assets)"
    echo "  clean          Clean build artifacts (dist/ etc)"
    echo "  help           Show this help"
    echo ""
    echo "Options:"
    echo "  --version VER  Version/tag to use for GitHub release (defaults to git describe)"
    echo "  --yes          Skip confirmation prompts (where applicable)"
    echo ""
    echo "Examples:"
    echo "  $0 package"
    echo "  $0 gh-release --version v1.2.3"
    echo "  $0 gh-full --version v1.2.3"
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

# (snapcraft-specific functions removed; releasing is GitHub-based now)

# Function to clean build artifacts
clean_build() {
    log_info "Cleaning build artifacts..."
    cd "$PROJECT_DIR"
    rm -rf dist/
    rm -rf build/
    log_success "Build artifacts cleaned"
}

# Main script logic
main() {
    local command=""
    local version=""
    local skip_confirm=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            package|gh-release|gh-full|clean|help)
                command="$1"
                shift
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
        package)
            package_release
            ;;
        gh-release)
            if [ -n "$version" ]; then
                tag="$version"
            else
                tag="$(git describe --tags --always)"
            fi
            gh_create_release "$tag"
            ;;
        gh-full)
            package_release
            if [ -n "$version" ]; then
                tag="$version"
            else
                tag="$(git describe --tags --always)"
            fi
            gh_create_release "$tag"
            ;;
        clean)
            clean_build
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