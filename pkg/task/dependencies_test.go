package task

import (
	"testing"

	"github.com/taha-shahid1/ecs-local/pkg/parser"
)

func TestBuildDependencyGraph(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{
			Name:  "db",
			Image: "postgres",
		},
		{
			Name:  "app",
			Image: "nginx",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "db", Condition: "START"},
			},
		},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	if len(graph.containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(graph.containers))
	}

	if len(graph.deps["app"]) != 1 {
		t.Errorf("Expected app to have 1 dependency, got %d", len(graph.deps["app"]))
	}

	if len(graph.deps["db"]) != 0 {
		t.Errorf("Expected db to have 0 dependencies, got %d", len(graph.deps["db"]))
	}
}

func TestGetStartOrder(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "c", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "b", Condition: "START"}}},
		{Name: "b", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "a", Condition: "START"}}},
		{Name: "a", Image: "test"},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	order := graph.getStartOrder()

	if len(order) != 3 {
		t.Fatalf("Expected 3 containers in order, got %d", len(order))
	}

	aIdx := indexOf(order, "a")
	bIdx := indexOf(order, "b")
	cIdx := indexOf(order, "c")

	if aIdx > bIdx {
		t.Errorf("Container a should start before b")
	}
	if bIdx > cIdx {
		t.Errorf("Container b should start before c")
	}
}

func TestDetectCycles(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{
			Name:  "a",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "b", Condition: "START"},
			},
		},
		{
			Name:  "b",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "a", Condition: "START"},
			},
		},
	}

	_, err := buildDependencyGraph(containers)
	if err == nil {
		t.Error("Expected error for circular dependency, got nil")
	}
}

func TestNonExistentDependency(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{
			Name:  "app",
			Image: "nginx",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "nonexistent", Condition: "START"},
			},
		},
	}

	_, err := buildDependencyGraph(containers)
	if err == nil {
		t.Error("Expected error for non-existent dependency, got nil")
	}
}

func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}
