# Bash Alias Manager

[![Get it from the Snap Store](https://snapcraft.io/en/dark/install.svg)](https://snapcraft.io/bash-alias-manager)

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
