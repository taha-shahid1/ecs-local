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
Multi-container task with Redis and Nginx, demonstrating container networking and dependencies.

```bash
ecs-dev run examples/multi-container.json
# Redis: 6379, Nginx: 8080
# The web container can access redis via hostname "redis"
```

### network-test.json
Simple network connectivity test with nginx server and alpine client.

```bash
ecs-dev run examples/network-test.json
# Watch the client container logs to see it successfully connect to the server container
ecs-dev logs client -f
```

### dependency-test.json
Tests container dependencies with COMPLETE condition.

```bash
ecs-dev run examples/dependency-test.json
# Init container runs first and completes, then app starts
```

### healthcheck-test.json
Tests container health checks.

```bash
ecs-dev run examples/healthcheck-test.json
# Nginx with health check that verifies HTTP endpoint
```

## Networking

Each task gets its own Docker bridge network, enabling container-to-container communication:

- All containers in a task are connected to the same network
- Containers can communicate using their container names as hostnames
- DNS resolution is handled automatically
- Networks are cleaned up when tasks are removed

Example: If you have containers named "web" and "redis" in the same task, the web container can connect to Redis at `redis:6379`.

## Container Dependencies

Control the startup order of containers using the `dependsOn` field:

- **START**: Wait for dependency to be running
- **COMPLETE**: Wait for dependency to exit (any exit code)
- **SUCCESS**: Wait for dependency to exit with code 0
- **HEALTHY**: Wait for dependency to pass health checks

Dependencies are validated at startup. Circular dependencies and missing containers are detected and rejected.

## Health Checks

Configure health checks to monitor container health:

```json
"healthCheck": {
  "command": ["CMD-SHELL", "wget -q --spider http://localhost || exit 1"],
  "interval": 10,
  "timeout": 5,
  "retries": 3,
  "startPeriod": 5
}
```

Health status is shown in the `ps` command output with color coding:
- Green: healthy
- Yellow: starting
- Red: unhealthy

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
