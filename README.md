# Crucible Hub

Registry server for managing and distributing Crucible resources with version
control and channel management.

## Overview

Hub provides a hierarchical registry API for managing:
- **Namespaces**: Top-level organizational units
- **Resources**: Named items within namespaces (widgets, services, etc.)
- **Versions**: Specific releases of resources with archives
- **Channels**: Named pointers to versions (e.g., `stable`, `latest`)

## Development

### Running Locally

```bash
# Build the binary
go build -o hub ./cmd/hub

# Run with default settings
./hub

# Run with custom configuration
PORT=8080 DB_PATH=./hub.db ARCHIVE_ROOT=./archives ./hub
```

### Environment Variables

- `PORT` - HTTP server port (default: `8080`)
- `DB_PATH` - SQLite database path (default: `./hub.db`)
- `ARCHIVE_ROOT` - Directory for storing archives (default: `./archives`)

## License

All rights reserved.
