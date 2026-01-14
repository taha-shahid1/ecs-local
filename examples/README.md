# Example Task Definitions

This directory contains example ECS task definition files that you can use to test `ecs-dev`.

## Files

### simple-nginx.json
Basic single-container task with nginx.

```bash
ecs-dev run examples/simple-nginx.json
# Visit http://localhost:8080
```

### multi-container.json
Multi-container task with Redis and Nginx, demonstrating dependencies.

```bash
ecs-dev run examples/multi-container.json
# Redis: 6379, Nginx: 8080
```

## Creating Your Own Task Definitions

Your task definition JSON should include:

### Required Fields
- `family` - A name for your task definition
- `containerDefinitions` - Array of container definitions
  - `name` - Container name
  - `image` - Docker image URI

### Common Optional Fields
- `memory` - Memory limit in MB
- `cpu` - CPU units (1024 = 1 vCPU)
- `memoryReservation` - Soft memory limit
- `portMappings` - Array of port mappings
  - `containerPort` - Container port
  - `hostPort` - Host port (defaults to containerPort)
  - `protocol` - tcp or udp (defaults to tcp)
- `environment` - Array of environment variables
  - `name` - Variable name
  - `value` - Variable value
- `command` - Override CMD
- `entryPoint` - Override ENTRYPOINT
- `dependsOn` - Container start order dependencies
- `links` - Container network links

See the [AWS ECS Task Definition documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_definitions.html) for more details.
