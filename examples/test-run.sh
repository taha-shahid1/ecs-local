#!/bin/bash
# Example script to test ecs-dev CLI

set -e

echo "Building ecs-dev..."
make build

echo ""
echo "Running simple nginx task..."
./bin/ecs-dev run examples/simple-nginx.json

echo ""
echo "Listing tasks..."
./bin/ecs-dev ps

echo ""
echo "Task is now running!"
echo "Visit http://localhost:8080 to see nginx"
echo ""
echo "To stop the task, use: ecs-dev stop <task-id>"
