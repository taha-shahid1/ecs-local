package task_test

import (
	"context"
	"fmt"
	"log"

	"github.com/taha-shahid1/ecs-local/pkg/docker"
	"github.com/taha-shahid1/ecs-local/pkg/parser"
	"github.com/taha-shahid1/ecs-local/pkg/task"
)

func ExampleManager_RunTask() {
	// Initialize Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to connect to Docker: %v", err)
	}
	defer dockerClient.Close()

	// Create task manager
	manager := task.NewManager(dockerClient)

	// Define a task (or parse from JSON file)
	taskDef := &parser.TaskDefinition{
		Family: "my-web-app",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:   "web",
				Image:  "nginx:alpine",
				Memory: 512,
				CPU:    256,
				PortMappings: []parser.PortMapping{
					{ContainerPort: 80, HostPort: 8080},
				},
				Environment: []parser.EnvironmentVariable{
					{Name: "ENV", Value: "production"},
				},
			},
		},
	}

	// Run the task
	ctx := context.Background()
	runningTask, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		log.Fatalf("Failed to run task: %v", err)
	}

	fmt.Printf("Task %s started with %d containers\n",
		runningTask.Family, len(runningTask.Containers))

	// Clean up
	defer func() {
		manager.StopTask(ctx, runningTask.ID)
		manager.RemoveTask(ctx, runningTask.ID)
	}()
}

func ExampleManager_ListTasks() {
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to connect to Docker: %v", err)
	}
	defer dockerClient.Close()

	manager := task.NewManager(dockerClient)

	// List all running tasks
	tasks := manager.ListTasks()

	fmt.Printf("Found %d running tasks\n", len(tasks))
	for _, t := range tasks {
		fmt.Printf("- %s (%s): %d containers\n",
			t.Family, t.Status, len(t.Containers))
	}
}
