# ecs-local

An open-source tool to simulate Amazon ECS (Elastic Container Service) locally using Docker.

## Overview

`ecs-local` allows you to run and test ECS task definitions locally without needing AWS infrastructure. It parses ECS task definition JSON files and orchestrates containers using the Docker SDK.

## Features (V1 - MVP)

- Parse and validate ECS task definition JSON
- Run containers using Docker SDK
- List running tasks and containers
- Stream and follow container logs
- Handle resource limits (CPU, memory)
- Support for container dependencies
- Port mappings and environment variables

## Prerequisites

- Go 1.20 or higher
- Docker installed and running
- Docker daemon accessible via Unix socket

## Installation

### From Source

```bash
git clone https://github.com/taha-shahid1/ecs-local.git
cd ecs-local

make install
# Or just: make build
```

Binary output: `./bin/ecs-dev`

## Usage

```bash
# Run a task from a task definition file
ecs-dev run task-definition.json

# List running tasks
ecs-dev ps
ecs-dev ps -a  # Show all tasks including stopped

# View logs
ecs-dev logs <container-name>
ecs-dev logs -f <container-name>  # Follow logs
ecs-dev logs --task <task-id>     # All containers in task
ecs-dev logs -f --task <task-id>  # Follow all containers

# Stop a task
ecs-dev stop <task-id>

# Stop all tasks
ecs-dev stop --all

# Remove a stopped task
ecs-dev rm <task-id>
ecs-dev rm -f <task-id>  # Force remove running task

# Show version
ecs-dev --version

# Show help
ecs-dev --help
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
      ]
    }
  ]
}
```
## Contributing

Contributions welcome.

## License

MIT License

## Author

Taha Shahid
