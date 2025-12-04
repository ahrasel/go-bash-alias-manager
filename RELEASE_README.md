# Bash Alias Manager - Release Script

This script automates the process of building, uploading, and releasing the Bash Alias Manager snap.

## Usage

```bash
./release.sh [COMMAND] [OPTIONS]
```

## Commands

- `build` - Build the snap locally
- `upload` - Upload snap to the Snap Store (requires login)
- `release` - Release snap to a channel (requires login)
- `status` - Check snap status in the store
- `clean` - Clean build artifacts
- `full` - Complete workflow: build → upload → release
- `help` - Show usage information

## Options

- `--channel CH` - Release channel (default: stable)
- `--version VER` - Specific version/revision to release
- `--yes` - Skip confirmation prompts

## Examples

```bash
# Build the snap
./release.sh build

# Build, upload, and release to stable
./release.sh full --channel stable

# Release specific revision to beta
./release.sh release --version 6 --channel beta

# Check snap status
./release.sh status

# Clean build artifacts
./release.sh clean
```

## Prerequisites

- `snapcraft` installed and logged in (`snapcraft login`)
- Project dependencies installed
- Proper permissions for snap store access

## Workflow

For a typical release:

1. Make your code changes
2. Update version in `snapcraft.yaml`
3. Run `./release.sh full --channel stable`
4. The script will handle building, uploading, and releasing

## Notes

- Classic confinement snaps require manual review by the Snap Store team
- The script includes error handling and confirmation prompts
- Use `--yes` flag for automated/CI usage
