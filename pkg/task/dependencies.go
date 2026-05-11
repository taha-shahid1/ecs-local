package task

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/taha-shahid1/ecs-local/pkg/parser"
)

// DependencyCondition represents a dependency with its condition
type DependencyCondition struct {
	ContainerName string
	Condition     string
}

// ContainerState tracks the state of a container for dependency monitoring
type ContainerState struct {
	Status    string // created, running, exited, dead
	ExitCode  int
	Health    string // none, starting, healthy, unhealthy
	LastCheck time.Time
	mu        sync.RWMutex
}

// dependencyGraph represents the dependency structure with conditions
type dependencyGraph struct {
	containers  map[string]*parser.ContainerDefinition
	deps        map[string][]DependencyCondition // container -> dependencies with conditions
	reverseDeps map[string][]string              // container -> dependents (for cascade failure)
}

func buildDependencyGraph(containerDefs []parser.ContainerDefinition) (*dependencyGraph, error) {
	graph := &dependencyGraph{
		containers:  make(map[string]*parser.ContainerDefinition),
		deps:        make(map[string][]DependencyCondition),
		reverseDeps: make(map[string][]string),
	}

	// Initialize all containers
	for i := range containerDefs {
		container := &containerDefs[i]
		graph.containers[container.Name] = container
		graph.deps[container.Name] = []DependencyCondition{}
		graph.reverseDeps[container.Name] = []string{}
	}

	// Build dependency graph with conditions
	for name, container := range graph.containers {
		for _, dep := range container.DependsOn {
			if _, exists := graph.containers[dep.ContainerName]; !exists {
				return nil, fmt.Errorf("container %s depends on non-existent container %s", name, dep.ContainerName)
			}

			// Add dependency with condition
			graph.deps[name] = append(graph.deps[name], DependencyCondition{
				ContainerName: dep.ContainerName,
				Condition:     dep.Condition,
			})

			// Build reverse dependency map for cascade failure handling
			graph.reverseDeps[dep.ContainerName] = append(graph.reverseDeps[dep.ContainerName], name)
		}
	}

	log.Printf("Validating dependency graph for cycles...")
	if err := graph.detectCycles(); err != nil {
		log.Printf("Cycle detection failed: %v", err)
		return nil, err
	}
	log.Printf("Dependency graph validation passed (no cycles detected)")

	return graph, nil
}

func (g *dependencyGraph) detectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for name := range g.containers {
		if err := g.detectCyclesHelper(name, visited, recStack); err != nil {
			return err
		}
	}

	return nil
}

func (g *dependencyGraph) detectCyclesHelper(name string, visited, recStack map[string]bool) error {
	if recStack[name] {
		return fmt.Errorf("circular dependency detected involving container %s", name)
	}

	if visited[name] {
		return nil
	}

	visited[name] = true
	recStack[name] = true

	for _, dep := range g.deps[name] {
		if err := g.detectCyclesHelper(dep.ContainerName, visited, recStack); err != nil {
			return err
		}
	}

	recStack[name] = false
	return nil
}

// getStartLevels returns containers grouped by dependency level using Kahn's algorithm
// Containers at the same level can be started in parallel
func (g *dependencyGraph) getStartLevels() [][]string {
	// Calculate in-degree for each container
	inDegree := make(map[string]int)
	for name := range g.containers {
		inDegree[name] = 0
	}

	for name, deps := range g.deps {
		inDegree[name] = len(deps)
	}

	levels := [][]string{}

	// Process levels iteratively
	for {
		// Find all containers with in-degree 0 (no unprocessed dependencies)
		currentLevel := []string{}
		for name, degree := range inDegree {
			if degree == 0 {
				currentLevel = append(currentLevel, name)
			}
		}

		if len(currentLevel) == 0 {
			break
		}

		levels = append(levels, currentLevel)

		// Remove current level containers and update in-degrees
		for _, name := range currentLevel {
			inDegree[name] = -1 // Mark as processed

			// Decrease in-degree for dependents
			for _, dependent := range g.reverseDeps[name] {
				if inDegree[dependent] > 0 {
					inDegree[dependent]--
				}
			}
		}
	}

	return levels
}

// GetDependents returns all containers that depend on the given container
func (g *dependencyGraph) GetDependents(containerName string) []string {
	return g.reverseDeps[containerName]
}

// GetDependencies returns all dependencies for a container with their conditions
func (g *dependencyGraph) GetDependencies(containerName string) []DependencyCondition {
	return g.deps[containerName]
}

func (m *Manager) waitForCondition(ctx context.Context, containerID string, condition string, waitingContainer string) error {
	switch condition {
	case "START":
		return m.waitForStart(ctx, containerID, waitingContainer)
	case "COMPLETE":
		return m.waitForComplete(ctx, containerID, waitingContainer)
	case "SUCCESS":
		return m.waitForSuccess(ctx, containerID, waitingContainer)
	case "HEALTHY":
		return m.waitForHealthy(ctx, containerID, waitingContainer)
	default:
		return fmt.Errorf("unknown condition: %s", condition)
	}
}

func (m *Manager) waitForStart(ctx context.Context, containerID string, waitingContainer string) error {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for container to start")
		case <-ticker.C:
			status, err := m.dockerClient.GetContainerStatus(ctx, containerID)
			if err != nil {
				return err
			}

			m.updateContainerState(containerID, status, -1, healthStatusNone)

			if status == containerStatusRunning {
				log.Printf("[%s] Dependency container started (START condition satisfied)", waitingContainer)
				return nil
			}

			if status == containerStatusExited || status == "dead" {
				inspect, err := m.dockerClient.GetContainerInspect(ctx, containerID)
				exitCode := -1
				if err == nil {
					exitCode = inspect.State.ExitCode
				}
				m.updateContainerState(containerID, status, exitCode, healthStatusNone)
				return fmt.Errorf("container exited before starting properly (exit code: %d)", exitCode)
			}
		}
	}
}

func (m *Manager) waitForComplete(ctx context.Context, containerID string, waitingContainer string) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for container to complete")
		case <-ticker.C:
			status, err := m.dockerClient.GetContainerStatus(ctx, containerID)
			if err != nil {
				return err
			}

			if status == containerStatusExited {
				inspect, err := m.dockerClient.GetContainerInspect(ctx, containerID)
				exitCode := -1
				if err == nil {
					exitCode = inspect.State.ExitCode
				}
				m.updateContainerState(containerID, status, exitCode, healthStatusNone)
				log.Printf("[%s] Dependency container completed (COMPLETE condition satisfied)", waitingContainer)
				return nil
			}

			m.updateContainerState(containerID, status, -1, healthStatusNone)
		}
	}
}

func (m *Manager) waitForSuccess(ctx context.Context, containerID string, waitingContainer string) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for container to succeed")
		case <-ticker.C:
			status, err := m.dockerClient.GetContainerStatus(ctx, containerID)
			if err != nil {
				return err
			}

			if status == containerStatusExited {
				inspect, err := m.dockerClient.GetContainerInspect(ctx, containerID)
				if err != nil {
					return err
				}

				exitCode := inspect.State.ExitCode
				m.updateContainerState(containerID, status, exitCode, healthStatusNone)

				if exitCode == 0 {
					log.Printf("[%s] Dependency container succeeded (SUCCESS condition satisfied)", waitingContainer)
					return nil
				}

				return fmt.Errorf("container exited with non-zero exit code: %d", exitCode)
			}

			m.updateContainerState(containerID, status, -1, healthStatusNone)
		}
	}
}

func (m *Manager) waitForHealthy(ctx context.Context, containerID string, waitingContainer string) error {
	if err := m.waitForStart(ctx, containerID, waitingContainer); err != nil {
		return err
	}

	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for container to become healthy")
		case <-ticker.C:
			health, err := m.dockerClient.GetContainerHealth(ctx, containerID)
			if err != nil {
				return err
			}

			status, _ := m.dockerClient.GetContainerStatus(ctx, containerID)
			m.updateContainerState(containerID, status, -1, health)

			if health == healthStatusNone {
				return fmt.Errorf("container does not have a health check configured")
			}

			if health == healthStatusHealthy {
				log.Printf("[%s] Dependency container is healthy (HEALTHY condition satisfied)", waitingContainer)
				return nil
			}

			if health == healthStatusUnhealthy {
				return fmt.Errorf("container is unhealthy")
			}
		}
	}
}

// updateContainerState updates the state of a container
func (m *Manager) updateContainerState(containerID string, status string, exitCode int, health string) {
	m.statesMu.Lock()
	defer m.statesMu.Unlock()

	state, exists := m.containerStates[containerID]
	if !exists {
		state = &ContainerState{}
		m.containerStates[containerID] = state
	}

	state.mu.Lock()
	state.Status = status
	if exitCode >= 0 {
		state.ExitCode = exitCode
	}
	if health != healthStatusNone {
		state.Health = health
	}
	state.LastCheck = time.Now()
	state.mu.Unlock()
}

// getContainerState returns the current state of a container
func (m *Manager) getContainerState(containerID string) *ContainerState {
	m.statesMu.RLock()
	defer m.statesMu.RUnlock()
	return m.containerStates[containerID]
}

// handleDependencyFailure stops dependent containers when a dependency fails
func (m *Manager) handleDependencyFailure(ctx context.Context, task *Task, failedContainer string, depGraph *dependencyGraph) {
	if !m.cascadeOnFailure {
		return
	}

	dependents := depGraph.GetDependents(failedContainer)
	if len(dependents) == 0 {
		return
	}

	log.Printf("[%s] Dependency failed, stopping %d dependent container(s): %v", failedContainer, len(dependents), dependents)
	fmt.Printf("⚠ Dependency %s failed, stopping dependent containers: %v\n", failedContainer, dependents)

	for _, dependent := range dependents {
		containerID, exists := task.Containers[dependent]
		if !exists {
			continue
		}

		// Check if container is already running
		status, err := m.dockerClient.GetContainerStatus(ctx, containerID)
		if err != nil || status != "running" {
			continue
		}

		err = m.dockerClient.StopContainer(ctx, containerID, 10)
		if err != nil {
			log.Printf("Warning: failed to stop dependent container %s: %v", dependent, err)
		} else {
			log.Printf("Stopped dependent container: %s", dependent)
			fmt.Printf("Stopped dependent container: %s\n", dependent)
		}
	}
}
