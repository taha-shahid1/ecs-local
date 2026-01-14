#!/bin/bash
# Example script to test ecs-dev CLI

set -e

echo "Building ecs-dev..."
make build

echo ""
echo "Running simple nginx task..."
TASK_OUTPUT=$(./bin/ecs-dev run examples/simple-nginx.json)
echo "$TASK_OUTPUT"

# Extract task ID from output
TASK_ID=$(echo "$TASK_OUTPUT" | grep "Task ID:" | awk '{print $3}')

echo ""
echo "Listing tasks..."
./bin/ecs-dev ps

echo ""
echo "Task is now running!"
echo "Visit http://localhost:8080 to see nginx"
echo ""
read -p "Press Enter to stop the task..."

echo ""
echo "Stopping task: $TASK_ID"
./bin/ecs-dev stop "$TASK_ID"

echo ""
echo "Listing tasks (stopped)..."
./bin/ecs-dev ps -a

echo ""
echo "Removing task: $TASK_ID"
./bin/ecs-dev rm "$TASK_ID"

echo ""
echo "✓ Test complete!"
