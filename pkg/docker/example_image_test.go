package docker_test

import (
	"context"
	"fmt"
	"log"

	"github.com/taha-shahid1/ecs-local/pkg/docker"
)

func ExampleClient_PullImage() {
	client, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pull an image with progress
	fmt.Println("Pulling nginx:alpine...")
	err = client.PullImage(ctx, "nginx:alpine", true)
	if err != nil {
		log.Fatalf("Failed to pull image: %v", err)
	}

	// Check if image exists
	exists, err := client.ImageExists(ctx, "nginx:alpine")
	if err != nil {
		log.Fatalf("Failed to check image: %v", err)
	}

	fmt.Printf("Image exists: %v\n", exists)
}

func ExampleClient_ListImages() {
	client, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	images, err := client.ListImages(ctx)
	if err != nil {
		log.Fatalf("Failed to list images: %v", err)
	}

	fmt.Printf("Found %d images\n", len(images))
	for _, img := range images {
		fmt.Println("-", img)
	}
}
