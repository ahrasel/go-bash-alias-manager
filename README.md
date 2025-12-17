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

## Quick Install / Uninstall / Run (short) ✅

Install (recommended — latest release):

```bash
sudo curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/releases/latest/download/install.sh | bash -s -- --desktop
```

Install to a custom location:

```bash
sudo curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/releases/latest/download/install.sh | bash -s -- --desktop --dest ~/.local/bin
```

Install from source (dev):

```bash
git clone https://github.com/ahrasel/go-bash-alias-manager.git
cd go-bash-alias-manager
go mod tidy
go build -o bash-alias-manager ./...
./bash-alias-manager
```

Uninstall (remove files):

```bash
sudo rm -f /usr/local/bin/bash-alias-manager || true
rm -f ~/.local/bin/bash-alias-manager || true
rm -f ~/.local/share/applications/bash-alias-manager.desktop || true
rm -f ~/.local/share/icons/hicolor/128x128/apps/bash-alias-manager.* || true
gtk-update-icon-cache -f -t ~/.local/share/icons/hicolor || true
update-desktop-database ~/.local/share/applications || true
```

Verify:

```bash
bash-alias-manager --help
bash-alias-manager --version
```

Update:

```bash
# Re-run the installer to update to the latest release
sudo curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/releases/latest/download/install.sh | bash -s -- --desktop

# Or install a specific version
sudo curl -fsSL https://github.com/ahrasel/go-bash-alias-manager/releases/download/v1.1.3/install.sh | bash -s -- --desktop
```

Quick troubleshooting:

- If `curl` returns 404, pin to a tag: `.../releases/download/v1.1.2/install.sh`.
- If the installer falls back to text parsing, install `jq` (`sudo apt-get install jq`).
- For build/linker issues install the GTK/OpenGL dev packages listed above.

If you'd like, I can also add a one-line install badge/link that points to the pinned release asset in the README.

# or build then run

go build -o bash-alias-manager ./...
./bash-alias-manager

````

Verify installation:

```bash
bash-alias-manager --help
bash-alias-manager --version
````

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
