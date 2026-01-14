package task

import (
	"context"
	"fmt"
	"time"

	"github.com/taha-shahid1/ecs-local/pkg/parser"
)

type dependencyGraph struct {
	containers map[string]*parser.ContainerDefinition
	deps       map[string][]string
}

func buildDependencyGraph(containerDefs []parser.ContainerDefinition) (*dependencyGraph, error) {
	graph := &dependencyGraph{
		containers: make(map[string]*parser.ContainerDefinition),
		deps:       make(map[string][]string),
	}

	for i := range containerDefs {
		container := &containerDefs[i]
		graph.containers[container.Name] = container
		graph.deps[container.Name] = []string{}
	}

	for name, container := range graph.containers {
		for _, dep := range container.DependsOn {
			if _, exists := graph.containers[dep.ContainerName]; !exists {
				return nil, fmt.Errorf("container %s depends on non-existent container %s", name, dep.ContainerName)
			}
			graph.deps[name] = append(graph.deps[name], dep.ContainerName)
		}
	}

	if err := graph.detectCycles(); err != nil {
		return nil, err
	}

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
		if err := g.detectCyclesHelper(dep, visited, recStack); err != nil {
			return err
		}
	}

	recStack[name] = false
	return nil
}

func (g *dependencyGraph) getStartOrder() []string {
	visited := make(map[string]bool)
	order := []string{}

	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		for _, dep := range g.deps[name] {
			visit(dep)
		}

		order = append(order, name)
	}

	for name := range g.containers {
		visit(name)
	}

	return order
}

func (m *Manager) waitForCondition(ctx context.Context, containerID string, condition string) error {
	switch condition {
	case "START":
		return m.waitForStart(ctx, containerID)
	case "COMPLETE":
		return m.waitForComplete(ctx, containerID)
	case "SUCCESS":
		return m.waitForSuccess(ctx, containerID)
	case "HEALTHY":
		return fmt.Errorf("HEALTHY condition not yet implemented")
	default:
		return fmt.Errorf("unknown condition: %s", condition)
	}
}

func (m *Manager) waitForStart(ctx context.Context, containerID string) error {
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

			if status == "running" {
				return nil
			}

			if status == "exited" || status == "dead" {
				return fmt.Errorf("container exited before starting properly")
			}
		}
	}
}

func (m *Manager) waitForComplete(ctx context.Context, containerID string) error {
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

			if status == "exited" {
				return nil
			}
		}
	}
}

func (m *Manager) waitForSuccess(ctx context.Context, containerID string) error {
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

			if status == "exited" {
				inspect, err := m.dockerClient.GetContainerInspect(ctx, containerID)
				if err != nil {
					return err
				}

				if inspect.State.ExitCode == 0 {
					return nil
				}

				return fmt.Errorf("container exited with non-zero exit code: %d", inspect.State.ExitCode)
			}
		}
	}
}
