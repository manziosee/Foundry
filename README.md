# Image Store - Container Image Management System

Minimal container image management system with FSM-based lifecycle and overlayfs snapshots.

## Architecture

- **FSM Engine**: Deterministic state transitions (NEW → DOWNLOADING → DOWNLOADED → UNPACKING → UNPACKED → STORED → ACTIVATING → ACTIVE)
- **Storage Backend**: Overlayfs snapshots for copy-on-write layers
- **Metadata DB**: SQLite with WAL mode for concurrent access
- **Blob Management**: HTTP/S3 download with checksum verification
- **Security**: Path traversal protection, atomic operations

## Quick Start

```bash
# Build
go build -o imgstore

# Start worker
./imgstore worker &

# Fetch image
./imgstore fetch myimage http://example.com/image.tar <sha256-checksum>

# Check status
./imgstore status myimage
```

## Test Setup

```bash
# Create test image
bash scripts/create-test-image.sh

# Start HTTP server
python3 -m http.server 8000 &

# Get checksum and fetch
CHECKSUM=$(sha256sum test-image.tar | cut -d' ' -f1)
./imgstore fetch testimg http://localhost:8000/test-image.tar $CHECKSUM
```

## Project Structure

```
├── cmd/manager/          # CLI manager with daemon mode
├── internal/fsm/         # State machine implementation
├── internal/storage/     # Overlayfs backend
├── migrations/           # SQLite schema
├── scripts/              # Test utilities
├── main.go              # Main CLI application
└── service.go           # Core service logic
```

## Storage Layout

- `store/blobs/` - Downloaded tarballs (by checksum)
- `store/images/` - Unpacked rootfs directories
- `store/overlays/` - Upper/work dirs for overlayfs
- `store/active/` - Active overlay mount points

## Requirements

- Go 1.21+
- Linux with overlayfs support
- Root privileges for overlay mounts

## FSM States

| State | Description |
|-------|-------------|
| NEW | Image registered, not downloaded |
| DOWNLOADING | Fetching blob from URL |
| DOWNLOADED | Blob present, checksum verified |
| UNPACKING | Extracting tarball |
| UNPACKED | Rootfs extracted successfully |
| STORED | Ready for activation |
| ACTIVATING | Creating overlay snapshot |
| ACTIVE | Overlay mounted and ready |
| FAILED | Terminal error state |

## Security Features

- SHA256 checksum validation
- Path traversal prevention during tar extraction
- Atomic file operations (download to .tmp, then rename)
- SQLite WAL mode for crash recovery
- Mount options: nodev,noexec,nosuid (when applicable)