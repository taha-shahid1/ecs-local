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

	if graph.deps["app"][0].ContainerName != "db" {
		t.Errorf("Expected app to depend on db, got %s", graph.deps["app"][0].ContainerName)
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

	levels := graph.getStartLevels()
	order := make([]string, 0, len(containers))
	for _, level := range levels {
		order = append(order, level...)
	}

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

func TestGetStartLevels(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "c", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "b", Condition: "START"}}},
		{Name: "b", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "a", Condition: "START"}}},
		{Name: "a", Image: "test"},
		{Name: "d", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "a", Condition: "START"}}},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	levels := graph.getStartLevels()

	if len(levels) != 3 {
		t.Fatalf("Expected 3 levels, got %d", len(levels))
	}

	// Level 1 should have "a" (no dependencies)
	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Errorf("Expected level 1 to have [a], got %v", levels[0])
	}

	// Level 2 should have "b" and "d" (both depend on "a")
	if len(levels[1]) != 2 {
		t.Errorf("Expected level 2 to have 2 containers, got %d", len(levels[1]))
	}
	hasB := false
	hasD := false
	for _, name := range levels[1] {
		if name == "b" {
			hasB = true
		}
		if name == "d" {
			hasD = true
		}
	}
	if !hasB || !hasD {
		t.Errorf("Expected level 2 to have [b, d], got %v", levels[1])
	}

	// Level 3 should have "c" (depends on "b")
	if len(levels[2]) != 1 || levels[2][0] != "c" {
		t.Errorf("Expected level 3 to have [c], got %v", levels[2])
	}
}

func TestGetDependents(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "a", Image: "test"},
		{Name: "b", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "a", Condition: "START"}}},
		{Name: "c", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "a", Condition: "START"}}},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	dependents := graph.GetDependents("a")
	if len(dependents) != 2 {
		t.Errorf("Expected a to have 2 dependents, got %d", len(dependents))
	}

	hasB := false
	hasC := false
	for _, name := range dependents {
		if name == "b" {
			hasB = true
		}
		if name == "c" {
			hasC = true
		}
	}
	if !hasB || !hasC {
		t.Errorf("Expected dependents [b, c], got %v", dependents)
	}
}

func TestGetDependencies(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "a", Image: "test"},
		{
			Name:  "b",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "a", Condition: "START"},
				{ContainerName: "a", Condition: "HEALTHY"},
			},
		},
		{
			Name:  "c",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "b", Condition: "SUCCESS"},
			},
		},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	deps := graph.GetDependencies("b")
	if len(deps) != 2 {
		t.Errorf("Expected b to have 2 dependencies, got %d", len(deps))
	}

	// Check that dependencies have correct conditions
	hasStart := false
	hasHealthy := false
	for _, dep := range deps {
		if dep.ContainerName == "a" {
			if dep.Condition == "START" {
				hasStart = true
			}
			if dep.Condition == "HEALTHY" {
				hasHealthy = true
			}
		}
	}
	if !hasStart || !hasHealthy {
		t.Errorf("Expected b to depend on a with START and HEALTHY conditions, got %v", deps)
	}

	depsC := graph.GetDependencies("c")
	if len(depsC) != 1 {
		t.Errorf("Expected c to have 1 dependency, got %d", len(depsC))
	}
	if depsC[0].ContainerName != "b" || depsC[0].Condition != "SUCCESS" {
		t.Errorf("Expected c to depend on b with SUCCESS condition, got %v", depsC[0])
	}
}

func TestGetDependencies_NoDependencies(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "a", Image: "test"},
		{Name: "b", Image: "test"},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	deps := graph.GetDependencies("a")
	if len(deps) != 0 {
		t.Errorf("Expected a to have 0 dependencies, got %d", len(deps))
	}
}

func TestGetStartLevels_ComplexGraph(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "a", Image: "test"},
		{Name: "b", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "a", Condition: "START"}}},
		{Name: "c", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "a", Condition: "START"}}},
		{Name: "d", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "b", Condition: "START"}, {ContainerName: "c", Condition: "START"}}},
		{Name: "e", Image: "test"},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	levels := graph.getStartLevels()

	if len(levels) != 3 {
		t.Fatalf("Expected 3 levels, got %d", len(levels))
	}

	// Level 1: a and e (no dependencies)
	if len(levels[0]) != 2 {
		t.Errorf("Expected level 1 to have 2 containers, got %d", len(levels[0]))
	}
	hasA := false
	hasE := false
	for _, name := range levels[0] {
		if name == "a" {
			hasA = true
		}
		if name == "e" {
			hasE = true
		}
	}
	if !hasA || !hasE {
		t.Errorf("Expected level 1 to have [a, e], got %v", levels[0])
	}

	// Level 2: b and c (both depend on a)
	if len(levels[1]) != 2 {
		t.Errorf("Expected level 2 to have 2 containers, got %d", len(levels[1]))
	}

	// Level 3: d (depends on b and c)
	if len(levels[2]) != 1 || levels[2][0] != "d" {
		t.Errorf("Expected level 3 to have [d], got %v", levels[2])
	}
}

func TestGetStartLevels_SingleContainer(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "a", Image: "test"},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	levels := graph.getStartLevels()

	if len(levels) != 1 {
		t.Fatalf("Expected 1 level, got %d", len(levels))
	}

	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Errorf("Expected level 1 to have [a], got %v", levels[0])
	}
}

func TestGetDependents_NoDependents(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "a", Image: "test"},
		{Name: "b", Image: "test"},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	dependents := graph.GetDependents("a")
	if len(dependents) != 0 {
		t.Errorf("Expected a to have 0 dependents, got %d", len(dependents))
	}
}

func TestGetDependents_MultipleDependents(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "base", Image: "test"},
		{Name: "a", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "base", Condition: "START"}}},
		{Name: "b", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "base", Condition: "START"}}},
		{Name: "c", Image: "test", DependsOn: []parser.ContainerDependency{{ContainerName: "base", Condition: "HEALTHY"}}},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	dependents := graph.GetDependents("base")
	if len(dependents) != 3 {
		t.Errorf("Expected base to have 3 dependents, got %d", len(dependents))
	}

	hasA := false
	hasB := false
	hasC := false
	for _, name := range dependents {
		if name == "a" {
			hasA = true
		}
		if name == "b" {
			hasB = true
		}
		if name == "c" {
			hasC = true
		}
	}
	if !hasA || !hasB || !hasC {
		t.Errorf("Expected dependents [a, b, c], got %v", dependents)
	}
}

func TestBuildDependencyGraph_WithAllConditions(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "init", Image: "test"},
		{
			Name:  "start-dep",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "init", Condition: "START"},
			},
		},
		{
			Name:  "complete-dep",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "init", Condition: "COMPLETE"},
			},
		},
		{
			Name:  "success-dep",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "init", Condition: "SUCCESS"},
			},
		},
		{
			Name:  "healthy-dep",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "init", Condition: "HEALTHY"},
			},
		},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	// Verify all conditions are preserved
	startDeps := graph.GetDependencies("start-dep")
	if len(startDeps) != 1 || startDeps[0].Condition != "START" {
		t.Errorf("Expected start-dep to have START condition, got %v", startDeps)
	}

	completeDeps := graph.GetDependencies("complete-dep")
	if len(completeDeps) != 1 || completeDeps[0].Condition != "COMPLETE" {
		t.Errorf("Expected complete-dep to have COMPLETE condition, got %v", completeDeps)
	}

	successDeps := graph.GetDependencies("success-dep")
	if len(successDeps) != 1 || successDeps[0].Condition != "SUCCESS" {
		t.Errorf("Expected success-dep to have SUCCESS condition, got %v", successDeps)
	}

	healthyDeps := graph.GetDependencies("healthy-dep")
	if len(healthyDeps) != 1 || healthyDeps[0].Condition != "HEALTHY" {
		t.Errorf("Expected healthy-dep to have HEALTHY condition, got %v", healthyDeps)
	}

	// Verify reverse dependencies are correct
	initDependents := graph.GetDependents("init")
	if len(initDependents) != 4 {
		t.Errorf("Expected init to have 4 dependents, got %d", len(initDependents))
	}
}

func TestBuildDependencyGraph_ReverseDependencies(t *testing.T) {
	containers := []parser.ContainerDefinition{
		{Name: "base", Image: "test"},
		{
			Name:  "middle",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "base", Condition: "START"},
			},
		},
		{
			Name:  "top1",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "middle", Condition: "START"},
			},
		},
		{
			Name:  "top2",
			Image: "test",
			DependsOn: []parser.ContainerDependency{
				{ContainerName: "middle", Condition: "START"},
			},
		},
	}

	graph, err := buildDependencyGraph(containers)
	if err != nil {
		t.Fatalf("Failed to build dependency graph: %v", err)
	}

	// Check reverse dependencies for base
	baseDependents := graph.GetDependents("base")
	if len(baseDependents) != 1 || baseDependents[0] != "middle" {
		t.Errorf("Expected base to have 1 dependent 'middle', got %v", baseDependents)
	}

	// Check reverse dependencies for middle
	middleDependents := graph.GetDependents("middle")
	if len(middleDependents) != 2 {
		t.Errorf("Expected middle to have 2 dependents, got %d", len(middleDependents))
	}
	hasTop1 := false
	hasTop2 := false
	for _, name := range middleDependents {
		if name == "top1" {
			hasTop1 = true
		}
		if name == "top2" {
			hasTop2 = true
		}
	}
	if !hasTop1 || !hasTop2 {
		t.Errorf("Expected middle to have dependents [top1, top2], got %v", middleDependents)
	}

	// Check reverse dependencies for top containers
	top1Dependents := graph.GetDependents("top1")
	if len(top1Dependents) != 0 {
		t.Errorf("Expected top1 to have 0 dependents, got %d", len(top1Dependents))
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
