# Conductor - Intent File Watcher

The conductor service watches the `handoff/` directory for new intent files and automatically triggers the porch-publisher to generate scaling patches.

## Overview

Conductor implements a file system watcher using `fsnotify` to monitor for new `intent-*.json` files. When a new intent file is detected, it automatically executes the porch-publisher to process the intent and generate the corresponding `scaling-patch.yaml`.

## Features

- Real-time monitoring of the `handoff/` directory
- Automatic triggering of porch-publisher for new intent files
- Correlation ID logging when present in intent JSON
- Configurable output directory via flag or environment variable
- Graceful shutdown on SIGINT/SIGTERM signals

## Usage

### Basic Usage

Run conductor with default settings (watches `handoff/` directory, outputs to `examples/packages/scaling`):

```bash
go run ./cmd/conductor
```

### Custom Configuration

Specify custom directories:

```bash
# Watch a different directory and specify output location
go run ./cmd/conductor -watch /path/to/watch -out /path/to/output

# Use environment variable for output directory
export CONDUCTOR_OUT_DIR=/custom/output/path
go run ./cmd/conductor
```

### Running with Intent Ingest

To use conductor with the intent-ingest service for automatic scaling patch updates:

1. Start the conductor in one terminal:
```bash
go run ./cmd/conductor
```

2. In another terminal, run intent-ingest to create new intent files:
```bash
# Example: create a scaling intent
echo '{"intent_type":"scaling","target":"my-app","namespace":"default","replicas":3,"correlation_id":"req-123"}' > handoff/intent-$(date +%Y%m%dT%H%M%S).json
```

3. Conductor will automatically detect the new file and trigger porch-publisher to generate `scaling-patch.yaml`

### Command Line Flags

- `-watch`: Directory to watch for intent-*.json files (default: "handoff")
- `-out`: Output directory for scaling patches (default: "examples/packages/scaling" or `$CONDUCTOR_OUT_DIR`)

### Environment Variables

- `CONDUCTOR_OUT_DIR`: Default output directory when `-out` flag is not specified

## Testing

Run the unit tests:

```bash
go test ./cmd/conductor
```

## Integration with CI/CD

Conductor can be integrated into your deployment pipeline to automatically generate Kubernetes patches based on intent files:

```bash
# Start conductor as a background service
go run ./cmd/conductor &
CONDUCTOR_PID=$!

# Your CI/CD pipeline creates intent files...
# Conductor automatically processes them

# Gracefully stop conductor
kill -TERM $CONDUCTOR_PID
```

## Architecture

The conductor follows a simple event-driven architecture:

1. **File Watcher**: Uses `fsnotify` to monitor the handoff directory
2. **Event Filter**: Filters for CREATE events on `intent-*.json` files
3. **Intent Processor**: Extracts correlation_id and triggers porch-publisher
4. **Command Executor**: Runs `go run ./cmd/porch-publisher` with appropriate arguments

## Logging

Conductor provides detailed logging including:
- Startup configuration (watched directory, output directory)
- Detection of new intent files
- Correlation IDs when present
- Success/failure of porch-publisher execution
- Graceful shutdown messages