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

If you want the installer to also install a desktop menu entry and icon (so the app appears in your desktop launcher), pass the `--desktop` flag. Example (install latest release and add desktop entry):

```bash
curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/raw/main/install.sh | bash -s -- --desktop --dest ~/.local/bin
```

Or using a specific release tag (more stable):

```bash
curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/raw/main/install.sh | bash -s -- --version v1.1.1 --desktop
```

If you prefer wget:

```bash
wget -qO- https://github.com/ahrasel/go-bash-alias-manager/raw/main/install.sh | bash -s -- --desktop
```

Troubleshooting

- "No release asset found for linux/amd64": the installer couldn't find a matching release asset. Confirm a release exists with an asset for your platform at https://github.com/ahrasel/go-bash-alias-manager/releases. You can also pass `--version <tag>` or `--url <asset-url>` to point to a specific release or asset.
- If the installer falls back to text parsing it prints a note recommending `jq` — install it for more reliable behavior: `sudo apt-get install jq`.
- If the `.desktop` entry doesn't appear in your application launcher immediately, check that the file was installed to `~/.local/share/applications/` (per-user) or `/usr/share/applications/` (system-wide). You can refresh the desktop database with `update-desktop-database ~/.local/share/applications` (if available) or log out/in.

Tip: If you run the installer piped from GitHub (curl | bash) and the script behaves unexpectedly, GitHub's raw content can be cached for a short time. To avoid caching issues, either specify a release with `--version v1.1.2` or fetch the installer at a specific commit:

```bash
curl -fsSL https://raw.githubusercontent.com/ahrasel/go-bash-alias-manager/<commit>/install.sh | bash -s -- --desktop
```

Manual desktop install

- Copy the desktop file and icon manually if needed:

```bash
mkdir -p ~/.local/share/applications ~/.local/share/icons/hicolor/128x128/apps
cp desktop/bash-alias-manager.desktop ~/.local/share/applications/
sed -i "s|Exec=bash-alias-manager|Exec=$(which bash-alias-manager)|" ~/.local/share/applications/bash-alias-manager.desktop
cp assets/icon.svg ~/.local/share/icons/hicolor/128x128/apps/bash-alias-manager.svg
gtk-update-icon-cache -f -t ~/.local/share/icons/hicolor || true
update-desktop-database ~/.local/share/applications || true
```

If you still have issues, run the installer script with debug output to see what is failing:

```bash
curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/raw/main/install.sh -o install.sh
bash -x install.sh --version v1.1.1 --desktop --dest ~/.local/bin
```

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

> ⚠️ **Platform support:** Because this project uses Fyne (desktop GUI) which depends on system graphics libraries, automated packaging currently produces Linux artifacts (x86_64) built from source on Linux runners. If `go build` fails locally due to Fyne/OpenGL dependencies, either:

- Build on a supported Linux machine (or CI runner) and push a release tag so the workflow produces artifacts, or
- Provide a prebuilt binary for the release in `dist/` before creating the GitHub release.

Note: CI now installs native GTK/OpenGL build dependencies on Ubuntu runners so Linux/Fyne builds should succeed automatically when you push a tag. If you see a linker error like "cannot find -lXxf86vm" or similar, install the missing X11/GL development package locally (on Debian/Ubuntu: `sudo apt-get install libxxf86vm-dev`) and retry; alternately provide a prebuilt binary in `dist/` before publishing.

Tip: The installer uses `jq` for reliable JSON parsing of GitHub release assets. If `jq` is not installed, the installer will fall back to a slower text-based parse and will print a note recommending `jq` (install with `sudo apt-get install jq`).

Contributions to make builds reproducible across macOS/Windows are welcome.
