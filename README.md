# Bash Alias Manager


https://github.com/ahrasel/go-bash-alias-manager

A desktop application built with Go and Fyne to manage your bash aliases on Linux.

## Features

- View all aliases from `~/.bash_aliases`
- Add new aliases (auto-saves and closes dialog)
- Edit existing aliases (auto-saves and closes dialog)
- Delete aliases (auto-saves after confirmation)
- Save changes back to the file (manual save button available)
- Backup aliases to GitHub Gist (cloud backup)
- Restore aliases from GitHub Gist
- Automatically ensures `~/.bashrc` sources `~/.bash_aliases`

## Requirements

- Go 1.21 or later
- Linux (designed for Linux desktop)

## Installation

1. Clone or download the project
2. Run `go mod tidy` to download dependencies
3. Run `go build` to build the application

## Usage

Run the executable:

```bash
./bash-alias-manager
```

The GUI will open, showing your current aliases. Select an alias from the list, then use the buttons to add, edit, or delete aliases. Changes are automatically saved to `~/.bash_aliases` after each operation. Click "Reload" to refresh the list from the file if needed.

### Cloud Backup

- **Backup**: Click "Backup" to upload your aliases to a private GitHub Gist. You'll be prompted for a GitHub Personal Access Token on first use (with `gist` scope). The Gist ID is stored locally for future updates.
- **Restore**: Click "Restore" to download and overwrite your local aliases from the cloud backup.

**Note**: Ensure your GitHub token has the `gist` permission. Create a token at https://github.com/settings/tokens with the "gist" scope selected.

## Notes

- If `~/.bash_aliases` doesn't exist, it will be created when you save.
- The app ensures that `~/.bashrc` includes a line to source `~/.bash_aliases` if it's not already there.
- Configuration (token and Gist ID) is stored in `~/.bash_alias_manager.json`.

## Install from GitHub Releases ✅

You can install the prebuilt binary from the GitHub Releases page using `curl` or `wget`:

Install latest release (recommended):

```bash
curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/raw/main/install.sh | bash
```

Or pipe the installer from the latest release download (more stable):

```bash
curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/releases/latest/download/install.sh | bash
```

The installer will detect your OS/architecture, download the appropriate asset, and install the `bash-alias-manager` binary to `/usr/local/bin` by default (you can pass `--dest` to install to another directory).

Verify the install:

```bash
bash-alias-manager --help
```

## Releasing 🔧

There are two supported release workflows:

- Local: build artifacts and create a GitHub release using the `release.sh` helper. Example:

```bash
# Build artifacts only
./release.sh package

# Create a GitHub release and upload artifacts using gh (or set GITHUB_TOKEN for API fallback)
./release.sh gh-release --version v1.2.3

# Build + publish
./release.sh gh-full --version v1.2.3
```

- Automated: push a tag `v1.2.3` to the repository and the GitHub Actions workflow will build and publish artifacts automatically.

The build script (`scripts/build_release.sh`) produces cross-compiled binaries and archives for common platforms and a SHA256 checksum file.

> ⚠️ **Platform support:** Because this project uses Fyne (desktop GUI) which depends on system graphics libraries, automated packaging currently produces Linux artifacts (x86_64) built from source on Linux runners. Cross-building macOS/Windows is not supported by the script at present; contributions to add reproducible builds for other platforms are welcome.

