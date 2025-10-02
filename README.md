# Image Store - Container Image Management System

Production-ready container image management system with FSM-based lifecycle, secure tar extraction, and overlayfs snapshots.

## Features

### Core Architecture
- **FSM Engine**: Deterministic state transitions with atomic updates
- **Storage Backend**: Overlayfs snapshots for copy-on-write layers
- **Metadata DB**: SQLite with WAL mode for concurrent access
- **Blob Management**: HTTP download with retry logic and caching
- **Security**: Comprehensive protection against malicious archives

### Advanced Capabilities
- **Retry Logic**: Automatic retry with exponential backoff (3 attempts)
- **Progress Tracking**: Real-time download progress monitoring
- **Blob Deduplication**: Cache-based storage to prevent re-downloads
- **Secure Extraction**: Protection against zip bombs, path traversal, symlink attacks
- **Resource Limits**: File size (100MB) and count (10K files) limits
- **Cleanup Management**: Automatic removal of unused blobs

## Quick Start

### Installation
```bash
# Clone repository
git clone https://github.com/manziosee/Foundry.git
cd Foundry

# Build
go build -o imgstore
```

### Basic Usage
```bash
# Start worker daemon
./imgstore worker &

# Fetch an image
./imgstore fetch myimage http://example.com/image.tar <sha256-checksum>

# Check status
./imgstore status myimage

# Cleanup unused blobs
./imgstore cleanup
```

## Complete Example

### 1. Create Test Environment
```bash
# Create test image (Windows)
mkdir test-rootfs\bin test-rootfs\etc
echo "echo Hello from container!" > test-rootfs\bin\hello.bat
echo "test:x:1000:1000:test:/:/bin/sh" > test-rootfs\etc\passwd
tar -cf test-image.tar -C test-rootfs .
rmdir /s /q test-rootfs

# Get checksum
certutil -hashfile test-image.tar SHA256
```

### 2. Start Local Server
```bash
# Python HTTP server
python -m http.server 8000

# Or Node.js
npx http-server -p 8000
```

### 3. Fetch and Activate
```bash
# Start worker
./imgstore worker &

# Fetch image (replace CHECKSUM with actual value)
./imgstore fetch testimg http://localhost:8000/test-image.tar CHECKSUM

# Monitor progress
./imgstore status testimg
```

## Architecture Deep Dive

### Project Structure
```
├── cmd/manager/              # CLI manager with daemon mode
│   ├── main.go              # CLI entry point
│   └── cleanup.go           # Blob cleanup functionality
├── internal/
│   ├── fsm/                 # Finite State Machine
│   │   └── fsm.go          # State definitions and transitions
│   ├── storage/             # Storage backends
│   │   └── overlay.go      # Overlayfs implementation
│   ├── downloader/          # HTTP download engine
│   │   └── downloader.go   # Retry logic and progress tracking
│   ├── extractor/           # Secure tar extraction
│   │   └── extractor.go    # Security-hardened extraction
│   └── cache/               # Blob caching system
│       └── cache.go        # Deduplication and cleanup
├── migrations/              # Database schema
│   └── 001_init.sql        # Initial SQLite schema
├── scripts/                 # Utilities and testing
│   ├── create-test-image.sh # Test image generator
│   └── create-malicious-tar.sh # Security test files
├── .github/workflows/       # CI/CD pipeline
│   └── ci.yml              # GitHub Actions workflow
├── main.go                  # Main CLI application
├── service.go              # Core service orchestration
└── README.md               # This file
```

### Storage Layout
```
store/
├── blobs/                   # Downloaded tarballs (by SHA256)
│   ├── abc123...def.tar    # Cached blob files
│   └── fed456...789.tar
├── images/                  # Unpacked rootfs directories
│   ├── myimage/rootfs/     # Extracted filesystem
│   └── testimg/rootfs/
├── overlays/               # Overlay filesystem layers
│   ├── myimage/
│   │   ├── upper/          # Read-write layer
│   │   └── work/           # Overlay work directory
│   └── testimg/
└── active/                 # Active overlay mount points
    ├── myimage/            # Live container filesystem
    └── testimg/
```

## State Machine (FSM)

### State Transitions
```
NEW → DOWNLOADING → DOWNLOADED → UNPACKING → UNPACKED → STORED → ACTIVATING → ACTIVE
 ↓         ↓            ↓           ↓          ↓         ↓          ↓
FAILED ←──┴────────────┴───────────┴──────────┴─────────┴──────────┘
```

### State Descriptions
| State | Description | Actions |
|-------|-------------|----------|
| NEW | Image registered, awaiting download | Queue for processing |
| DOWNLOADING | HTTP download in progress | Retry on failure, track progress |
| DOWNLOADED | Blob cached, checksum verified | Mark blob as used |
| UNPACKING | Secure tar extraction running | Validate paths, check limits |
| UNPACKED | Rootfs extracted successfully | Prepare for storage |
| STORED | Ready for activation | Create overlay directories |
| ACTIVATING | Creating overlay snapshot | Mount overlayfs |
| ACTIVE | Image ready for use | Available for containers |
| FAILED | Terminal error state | Cleanup partial files |

## Security Model

### Download Security
- **Checksum Validation**: SHA256 verification during download
- **Atomic Operations**: Download to `.tmp`, rename on success
- **Retry Logic**: Exponential backoff with 3 attempts
- **Context Cancellation**: Graceful shutdown support

### Extraction Security
- **Path Traversal Protection**: Blocks `../` and absolute paths
- **Symlink Validation**: Ensures symlinks stay within bounds
- **File Size Limits**: 100MB per file, 10K files maximum
- **Permission Sanitization**: Limits to 0755 (exec) or 0644 (regular)
- **Archive Bomb Protection**: Memory-efficient streaming extraction

### Storage Security
- **Isolation**: Each image in separate overlay namespace
- **Cleanup**: Automatic removal of failed/unused artifacts
- **Database Integrity**: SQLite WAL mode with atomic transactions

## API Reference

### CLI Commands
```bash
# Image Management
./imgstore fetch <name> <url> <checksum>  # Download and process image
./imgstore status <name>                  # Check image state
./imgstore worker                         # Start processing daemon

# Maintenance
./imgstore cleanup                        # Remove unused blobs
./imgstore list                          # List all images (planned)
```

### Manager CLI (Advanced)
```bash
# Daemon mode
./manager --daemon --db ./custom.db --store ./custom-store

# Direct commands
./manager fetch myimage http://example.com/image.tar abc123
./manager status myimage
./manager cleanup
```

## Development

### Requirements
- **Go 1.21+** with CGO enabled (for SQLite)
- **Linux/Windows** with tar support
- **Root privileges** for overlay mounts (Linux)

### Building
```bash
# Development build
go build -o imgstore

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o imgstore-linux
GOOS=windows GOARCH=amd64 go build -o imgstore.exe
GOOS=darwin GOARCH=amd64 go build -o imgstore-darwin
```

### Testing
```bash
# Run tests
go test ./...

# Test with malicious archives (security validation)
bash scripts/create-malicious-tar.sh
# Test extraction security manually
```

### CI/CD
GitHub Actions automatically:
- Runs tests on push/PR
- Builds for multiple platforms
- Validates code quality
- Checks security compliance

## Production Deployment

### System Requirements
- **CPU**: 2+ cores recommended
- **Memory**: 1GB+ RAM
- **Storage**: SSD recommended for blob cache
- **Network**: Stable internet for image downloads

### Configuration
```bash
# Production setup
export IMGSTORE_DB_PATH=/var/lib/imgstore/store.db
export IMGSTORE_STORE_PATH=/var/lib/imgstore/store
export IMGSTORE_LOG_LEVEL=info

# Start as systemd service
sudo systemctl enable imgstore-worker
sudo systemctl start imgstore-worker
```

### Monitoring
- **Logs**: Structured logging with timestamps
- **Metrics**: Download progress, success/failure rates
- **Health**: Database connectivity, storage space

## Roadmap

### Completed ✅
- [x] FSM-based image lifecycle
- [x] HTTP download with retry logic
- [x] Secure tar extraction
- [x] Blob caching and deduplication
- [x] Overlayfs storage backend
- [x] CLI interface and daemon mode

### In Progress 🚧
- [ ] DeviceMapper thin-pool backend
- [ ] REST API endpoints
- [ ] Web dashboard interface

### Planned 📋
- [ ] Image signing and verification
- [ ] Multi-architecture support
- [ ] Prometheus metrics
- [ ] Container runtime integration
- [ ] Kubernetes operator

## Contributing

1. **Fork** the repository
2. **Create** feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** changes (`git commit -m 'Add amazing feature'`)
4. **Push** to branch (`git push origin feature/amazing-feature`)
5. **Open** Pull Request

### Development Workflow
- Use descriptive commit messages
- Add tests for new features
- Update documentation
- Follow Go best practices
- Ensure CI passes

## License

MIT License - see LICENSE file for details.

## Support

- **Issues**: GitHub Issues for bug reports
- **Discussions**: GitHub Discussions for questions
- **Security**: Email security@example.com for vulnerabilities