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

## Install, Uninstall, and Run (users) ✅

This section explains the recommended ways to install (from GitHub Releases), uninstall, and run the application locally from the repository.

### Install — GitHub Releases (recommended) 🔽

- Install the latest release (installer will detect OS/arch and install to `/usr/local/bin` by default):

```bash
curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/releases/latest/download/install.sh -o install.sh && bash install.sh --desktop
```

- Pipe installer directly (pin to a tag or commit for stability):

```bash
curl -fsSL https://raw.githubusercontent.com/ahrasel/go-bash-alias-manager/<tag-or-commit>/install.sh | bash -s -- --desktop --dest ~/.local/bin
```

- `wget` equivalent:

```bash
wget -qO- https://github.com/ahrasel/go-bash-alias-manager/releases/latest/download/install.sh | bash -s -- --desktop
```

Options:
- `--dest <path>` — install binary to a custom directory (e.g., `~/.local/bin`).
- `--desktop` — also install a `.desktop` menu entry and icon for desktop launchers.

Tip: prefer the release download URL above (`/releases/latest/download/install.sh`) or pin a **tag/commit** to avoid transient raw-content cache issues on GitHub.

### Install — From source (developer or maintainers) 🛠️

If you want to build locally from the repository:

```bash
git clone https://github.com/ahrasel/go-bash-alias-manager.git
cd go-bash-alias-manager
go mod tidy
go build -o bash-alias-manager ./...
./bash-alias-manager
```

Or to install into your Go bin:

```bash
go install github.com/ahrasel/go-bash-alias-manager@latest
# then run: $GOBIN/bash-alias-manager or $HOME/go/bin/bash-alias-manager
```

Notes:
- Requires Go 1.21+. Building the GUI (Fyne) requires native graphics libraries on Linux (GTK/OpenGL). If `go build` fails with a linker error like `-lXxf86vm` install the corresponding dev package (Debian/Ubuntu: `sudo apt-get install libxxf86vm-dev libcairo2-dev libpango1.0-dev libgtk-3-dev libgl1-mesa-dev`).

### Uninstall — Remove binary, desktop entry & icon 🧹

Run the commands below adapted to how you installed (system vs user):

```bash
# Remove binary
rm -f /usr/local/bin/bash-alias-manager          # system install
rm -f ~/.local/bin/bash-alias-manager            # per-user install

# Remove desktop files and icons
rm -f ~/.local/share/applications/bash-alias-manager.desktop
rm -f ~/.local/share/icons/hicolor/128x128/apps/bash-alias-manager.svg
rm -f ~/.local/share/icons/hicolor/128x128/apps/bash-alias-manager.png || true

# Update icon/desktop caches (if available)
gtk-update-icon-cache -f -t ~/.local/share/icons/hicolor || true
update-desktop-database ~/.local/share/applications || true

# Remove local configuration (optional)
rm -f ~/.bash_alias_manager.json
```

### Run locally from Git (development) ▶️

To run the app directly from source without installing:

```bash
git clone https://github.com/ahrasel/go-bash-alias-manager.git
cd go-bash-alias-manager
go mod tidy
go run ./...
# or build then run
go build -o bash-alias-manager ./...
./bash-alias-manager
```

Verify installation:

```bash
bash-alias-manager --help
bash-alias-manager --version
```

### Troubleshooting ⚠️
- If the installer prints "No release asset found for linux/amd64" check that a release with the appropriate artifact exists on the Releases page and try `--version <tag>` or `--url <asset-url>`.
- The installer uses `jq` for robust JSON parsing. If `jq` is not installed, the installer will fall back to text parsing and will print a note recommending `jq` (install with `sudo apt-get install jq`).
- If you see linker errors building from source related to missing native libs (Fyne/OpenGL), install the platform dev packages listed above and retry.

If you want, I can also add a short one-line install link to the README (a pinned release `install.sh` asset), so `curl | bash` always picks the fixed script — tell me if you'd like me to upload `install.sh` as a release asset for the latest tag.

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
