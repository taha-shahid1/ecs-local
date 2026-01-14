# ecs-local

An open-source tool to simulate Amazon ECS (Elastic Container Service) locally using Docker.

## Overview

`ecs-local` allows you to run and test ECS task definitions locally without needing AWS infrastructure. It parses ECS task definition JSON files and orchestrates containers using the Docker SDK.

## Features (V1 - MVP)

- 🚀 Parse and validate ECS task definition JSON
- 🐳 Run containers using Docker SDK
- 📊 List running tasks and containers
- 📝 Stream and follow container logs
- ⚙️ Handle resource limits (CPU, memory)
- 🔗 Support for container dependencies
- 🌐 Port mappings and environment variables

## Prerequisites

- Go 1.20 or higher
- Docker installed and running
- Docker daemon accessible via Unix socket

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/tahashahid/ecs-local.git
cd ecs-local

# Build and install
make install

# Or just build
make build
```

The binary will be available at `./bin/ecs-dev`

## Usage

```bash
# Run a task from a task definition file
ecs-dev run task-definition.json

# List running tasks
ecs-dev ps

# View container logs
ecs-dev logs <container-name>

# Follow container logs
ecs-dev logs -f <container-name>

# Stop a task
ecs-dev stop <task-name>

# Stop all tasks
ecs-dev stop --all

# Remove a stopped task
ecs-dev rm <task-name>

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

## Roadmap

- [x] V1: MVP - Basic task execution and container orchestration
- [ ] V2: Service mode with health checks and load balancing
- [ ] V3: Task placement strategies and constraints
- [ ] V4: Integration with AWS services (Secrets Manager, Parameter Store)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Author

Taha Shahid
