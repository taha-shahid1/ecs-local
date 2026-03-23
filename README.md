# ecs-local

[![CI](https://github.com/taha-shahid1/ecs-local/actions/workflows/ci.yml/badge.svg)](https://github.com/taha-shahid1/ecs-local/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go)](https://golang.org/)

A CLI tool to run and test Amazon ECS task definitions locally using Docker.

## Why ecs-local?

Testing ECS tasks shouldn't require deploying to AWS. `ecs-local` simulates ECS locally, allowing you to:
- Test task definitions before deploying to production
- Develop multi-container applications with realistic networking
- Debug container dependencies and health checks locally
- Iterate faster without AWS costs or complexity

## Features

- **Task orchestration** - Parse and run ECS task definitions locally
- **Multi-container networking** - Isolated Docker networks with DNS-based service discovery
- **Container dependencies** - Smart startup ordering with parallel execution
- **Dependency conditions** - Support for START, COMPLETE, SUCCESS, and HEALTHY conditions
- **Cascade failure handling** - Automatically stop dependent containers when dependencies fail
- **Health checks** - Monitor container health with configurable checks
- **Volume management** - Named volumes and bind mounts with automatic lifecycle management
- **Interactive exec** - Execute commands in running containers
- **Log streaming** - View and follow container logs in real-time

## Prerequisites

- Go 1.20 or higher
- Docker installed and running
- Docker daemon accessible via Unix socket

## Installation

```bash
git clone https://github.com/taha-shahid1/ecs-local.git
cd ecs-local
make install
```

The binary will be installed to `$GOPATH/bin/ecs-dev`. Make sure `$GOPATH/bin` is in your PATH.

Alternatively, build without installing:
```bash
make build
./bin/ecs-dev --version
```

## Quick Start

```bash
# Run a simple nginx task
ecs-dev run examples/simple-nginx.json

# List running tasks
ecs-dev ps

# View logs
ecs-dev logs nginx -f

# Stop the task
ecs-dev stop <task-id>
```

## Usage

### Running Tasks

```bash
# Run a task definition
ecs-dev run task-definition.json

# Disable cascade failure (keep dependents running if dependency fails)
ecs-dev run --no-cascade task-definition.json
```

### Viewing Tasks

```bash
# List running tasks
ecs-dev ps

# Show all tasks (including stopped)
ecs-dev ps -a

# Show detailed dependency information
ecs-dev ps --deps
```

### Logs

```bash
# View container logs
ecs-dev logs <container-name>

# Follow logs in real-time
ecs-dev logs -f <container-name>

# View all containers in a task
ecs-dev logs --task <task-id>

# Follow all containers in a task
ecs-dev logs -f --task <task-id>
```

### Executing Commands

```bash
# Run a command in a container
ecs-dev exec <container-name> whoami

# Use -- for commands with flags
ecs-dev exec <container-name> -- ls -la /app

# Interactive shell
ecs-dev exec -i <container-name> sh

# Execute as specific user
ecs-dev exec -u nginx <container-name> whoami
```

### Dependency Information

```bash
# Show dependency graph for all tasks
ecs-dev deps

# Show dependency graph for specific task
ecs-dev deps <task-id>
```

### Stopping and Removing Tasks

```bash
# Stop a task
ecs-dev stop <task-id>

# Stop all tasks
ecs-dev stop --all

# Remove a stopped task
ecs-dev rm <task-id>

# Force remove running task
ecs-dev rm -f <task-id>
```

## Project Structure

```
ecs-local/
├── cmd/
│   └── ecs-dev/          # CLI entry point
│       └── main.go
├── pkg/                   # Public packages
│   ├── config/           # Configuration management
│   ├── docker/           # Docker SDK wrapper
│   ├── parser/           # Task definition parser
│   └── task/             # Task lifecycle management
├── tests/
│   └── integration/      # Integration tests
├── go.mod
├── Makefile
└── README.md
```

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter (requires golangci-lint)
make lint

# Download dependencies
make deps

# Clean build artifacts
make clean
```

## Task Definition Example

Create a `task-definition.json` file:

```json
{
  "family": "my-app",
  "containerDefinitions": [
    {
      "name": "web",
      "image": "nginx:latest",
      "memory": 512,
      "cpu": 256,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "hostPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "ENV",
          "value": "development"
        }
      ],
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost/ || exit 1"],
        "interval": 10,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 5
      }
    }
  ]
}
```

More examples in the [`examples/`](examples/) directory:
- `simple-nginx.json` - Basic single-container task
- `multi-container.json` - Multi-container with Redis and Nginx
- `dependency-test.json` - Container dependencies
- `healthcheck-test.json` - Health check configuration
- `volume-test.json` - Volume sharing between containers

See the [examples README](examples/README.md) for detailed documentation.
## How It Works

1. **Parse** - Reads and validates ECS task definition JSON
2. **Create** - Creates Docker containers with proper configuration
3. **Network** - Sets up isolated Docker network with DNS resolution
4. **Dependencies** - Analyzes dependency graph and determines start order
5. **Start** - Launches containers in parallel by dependency level
6. **Monitor** - Tracks container state, health, and dependency conditions

Containers at the same dependency level start in parallel for optimal performance. If a dependency fails, dependent containers are automatically stopped (configurable with `--no-cascade`).

## Roadmap

- [ ] GitHub Actions for automated releases
- [ ] Pre-built binaries for multiple platforms
- [ ] Homebrew formula for easier installation
- [ ] Support for task roles and IAM
- [ ] Secrets management integration
- [ ] ECS service simulation (auto-restart, desired count)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

**Taha Shahid** - [GitHub](https://github.com/taha-shahid1)

## Acknowledgments

- Inspired by the need for faster ECS development workflows
- Built with the [Docker Go SDK](https://github.com/docker/docker)
- CLI powered by [Cobra](https://github.com/spf13/cobra)
