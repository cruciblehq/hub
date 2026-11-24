# Crucible Hub

Registry server for managing and distributing Crucible resources with version
control and channel management.

## Development

### Building the Docker Image

```bash
# Build multi-platform OCI image
./scripts/build.sh
```

The build script creates a universal Docker image supporting both `linux/amd64`
and `linux/arm64`, outputting to `dist/image.tar` in OCI format.

### Environment Variables

- `PORT` - HTTP server port (default: `8080`)
- `DB_PATH` - SQLite database path (default: `./hub.db`)
- `ARCHIVE_ROOT` - Directory for storing archives (default: `./archives`)

## License

All rights reserved.
