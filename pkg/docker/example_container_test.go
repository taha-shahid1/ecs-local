package docker_test

import (
	"context"
	"fmt"
	"log"

	"github.com/taha-shahid1/ecs-local/pkg/docker"
)

func ExampleClient_CreateContainer() {
	client, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pull the image first
	err = client.PullImage(ctx, "nginx:alpine", false)
	if err != nil {
		log.Fatalf("Failed to pull image: %v", err)
	}

	// Create container with configuration
	config := docker.ContainerConfig{
		Name:  "my-web-server",
		Image: "nginx:alpine",
		Env: []string{
			"NGINX_HOST=localhost",
			"NGINX_PORT=80",
		},
		PortMap: map[string]string{
			"80": "8080", // container:host
		},
		Memory: 512 * 1024 * 1024, // 512MB
		CPU:    512,               // CPU shares
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	fmt.Printf("Created container: %s\n", containerID[:12])

	// Start the container
	err = client.StartContainer(ctx, containerID)
	if err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}

	fmt.Println("Container started successfully")

	// Clean up
	defer func() {
		client.StopContainer(ctx, containerID, 10)
		client.RemoveContainer(ctx, containerID, true)
	}()
}

func Example_containerLifecycle() {
	client, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Ensure image exists
	client.PullImage(ctx, "alpine:latest", false)

	// Create
	config := docker.ContainerConfig{
		Name:  "test-container",
		Image: "alpine:latest",
		Cmd:   []string{"echo", "Hello from ECS local!"},
	}

	id, _ := client.CreateContainer(ctx, config)
	fmt.Println("Created:", id[:12])

	// Start
	client.StartContainer(ctx, id)
	fmt.Println("Started")

	// Check status
	status, _ := client.GetContainerStatus(ctx, id)
	fmt.Println("Status:", status)

	// Stop
	client.StopContainer(ctx, id, 5)
	fmt.Println("Stopped")

	// Remove
	client.RemoveContainer(ctx, id, false)
	fmt.Println("Removed")
}
