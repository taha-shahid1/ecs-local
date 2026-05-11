package parser

import (
	"fmt"
	"path/filepath"
	"testing"
)

func ExampleParseTaskDefinition() {
	taskDef, err := ParseTaskDefinition("../../examples/simple-nginx.json")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Task Family: %s\n", taskDef.Family)
	fmt.Printf("Number of Containers: %d\n", len(taskDef.ContainerDefinitions))

	for _, container := range taskDef.ContainerDefinitions {
		fmt.Printf("Container: %s (Image: %s)\n", container.Name, container.Image)
	}
	// Output:
	// Task Family: simple-nginx
	// Number of Containers: 1
	// Container: nginx (Image: nginx:latest)
}

// TestExampleFiles ensures the example task definitions are valid
func TestExampleFiles(t *testing.T) {
	exampleFiles := []string{
		"../../examples/simple-nginx.json",
		"../../examples/multi-container.json",
	}

	for _, file := range exampleFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			taskDef, err := ParseTaskDefinition(file)
			if err != nil {
				t.Errorf("failed to parse %s: %v", file, err)
				return
			}

			if taskDef.Family == "" {
				t.Error("task family should not be empty")
			}

			if len(taskDef.ContainerDefinitions) == 0 {
				t.Error("should have at least one container definition")
			}

			for _, container := range taskDef.ContainerDefinitions {
				if container.Name == "" {
					t.Error("container name should not be empty")
				}
				if container.Image == "" {
					t.Error("container image should not be empty")
				}
			}
		})
	}
}
