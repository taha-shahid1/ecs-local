package docker_test

import (
	"context"
	"fmt"
	"log"

	"github.com/taha-shahid1/ecs-local/pkg/docker"
)

func ExampleNewClient() {
	client, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	version, err := client.Version(ctx)
	if err != nil {
		log.Fatalf("Failed to get Docker version: %v", err)
	}

	fmt.Printf("Connected to Docker: %s\n", version)
}
