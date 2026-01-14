package task

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/taha-shahid1/ecs-local/pkg/docker"
	"github.com/taha-shahid1/ecs-local/pkg/parser"
)

// Task represents a running ECS task
type Task struct {
	ID          string
	Family      string
	Containers  map[string]string // containerName -> containerID
	NetworkID   string
	NetworkName string
	Status      string
	StartedAt   time.Time
	Definition  *parser.TaskDefinition
}

// Manager handles task lifecycle
type Manager struct {
	dockerClient *docker.Client
	tasks        map[string]*Task // taskID -> Task
}

// NewManager initializes a task manager
func NewManager(dockerClient *docker.Client) *Manager {
	return &Manager{
		dockerClient: dockerClient,
		tasks:        make(map[string]*Task),
	}
}

// RunTask creates and starts containers for a task definition
func (m *Manager) RunTask(ctx context.Context, taskDef *parser.TaskDefinition) (*Task, error) {
	taskID := generateTaskID(taskDef.Family)
	networkName := fmt.Sprintf("ecs-local-%s", taskID)

	task := &Task{
		ID:          taskID,
		Family:      taskDef.Family,
		Containers:  make(map[string]string),
		NetworkName: networkName,
		Status:      "PROVISIONING",
		StartedAt:   time.Now(),
		Definition:  taskDef,
	}

	m.tasks[taskID] = task

	networkID, err := m.dockerClient.CreateNetwork(ctx, networkName)
	if err != nil {
		task.Status = "FAILED"
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	task.NetworkID = networkID
	fmt.Printf("Created network: %s (%s)\n", networkName, networkID[:12])

	defer func() {
		if task.Status == "FAILED" {
			m.cleanupFailedTask(ctx, task)
		}
	}()

	// Pull all images first
	for _, containerDef := range taskDef.ContainerDefinitions {
		fmt.Printf("Pulling image: %s\n", containerDef.Image)
		err := m.dockerClient.PullImage(ctx, containerDef.Image, true)
		if err != nil {
			task.Status = "FAILED"
			return nil, fmt.Errorf("failed to pull image %s: %w", containerDef.Image, err)
		}
	}

	// Create and start containers
	for _, containerDef := range taskDef.ContainerDefinitions {
		containerConfig := convertToDockerConfig(taskID, containerDef)

		containerID, err := m.dockerClient.CreateContainer(ctx, containerConfig)
		if err != nil {
			task.Status = "FAILED"
			return nil, fmt.Errorf("failed to create container %s: %w", containerDef.Name, err)
		}

		task.Containers[containerDef.Name] = containerID

		fmt.Printf("Created container: %s (%s)\n", containerDef.Name, containerID[:12])

		aliases := []string{containerDef.Name}
		err = m.dockerClient.ConnectContainerToNetwork(ctx, networkID, containerID, aliases)
		if err != nil {
			task.Status = "FAILED"
			return nil, fmt.Errorf("failed to connect container %s to network: %w", containerDef.Name, err)
		}
		fmt.Printf("Connected %s to network %s\n", containerDef.Name, networkName)
	}

	// Start containers
	for name, containerID := range task.Containers {
		err := m.dockerClient.StartContainer(ctx, containerID)
		if err != nil {
			task.Status = "FAILED"
			return nil, fmt.Errorf("failed to start container %s: %w", name, err)
		}
		fmt.Printf("Started container: %s\n", name)
	}

	task.Status = "RUNNING"
	return task, nil
}

// StopTask stops all containers in a task
func (m *Manager) StopTask(ctx context.Context, taskID string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = "STOPPING"

	for name, containerID := range task.Containers {
		err := m.dockerClient.StopContainer(ctx, containerID, 10)
		if err != nil {
			fmt.Printf("Warning: failed to stop container %s: %v\n", name, err)
		} else {
			fmt.Printf("Stopped container: %s\n", name)
		}
	}

	task.Status = "STOPPED"
	return nil
}

// RemoveTask removes all containers in a task
func (m *Manager) RemoveTask(ctx context.Context, taskID string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	for name, containerID := range task.Containers {
		err := m.dockerClient.RemoveContainer(ctx, containerID, true)
		if err != nil {
			fmt.Printf("Warning: failed to remove container %s: %v\n", name, err)
		} else {
			fmt.Printf("Removed container: %s\n", name)
		}
	}

	if task.NetworkID != "" {
		err := m.dockerClient.RemoveNetwork(ctx, task.NetworkID)
		if err != nil {
			fmt.Printf("Warning: failed to remove network %s: %v\n", task.NetworkName, err)
		} else {
			fmt.Printf("Removed network: %s\n", task.NetworkName)
		}
	}

	delete(m.tasks, taskID)
	return nil
}

// GetTask returns a task by ID
func (m *Manager) GetTask(taskID string) (*Task, error) {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// ListTasks returns all tasks
func (m *Manager) ListTasks() []*Task {
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// convertToDockerConfig converts a parser.ContainerDefinition to docker.ContainerConfig
func convertToDockerConfig(taskID string, containerDef parser.ContainerDefinition) docker.ContainerConfig {
	config := docker.ContainerConfig{
		Name:  fmt.Sprintf("%s-%s", taskID, containerDef.Name),
		Image: containerDef.Image,
	}

	// Environment variables
	for _, env := range containerDef.Environment {
		config.Env = append(config.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	// Command and entrypoint
	if len(containerDef.Command) > 0 {
		config.Cmd = containerDef.Command
	}
	if len(containerDef.EntryPoint) > 0 {
		config.Entrypoint = containerDef.EntryPoint
	}

	// Memory (convert MB to bytes)
	memory := containerDef.GetEffectiveMemory()
	if memory > 0 {
		config.Memory = int64(memory) * 1024 * 1024
	}

	// CPU
	if containerDef.CPU > 0 {
		config.CPU = int64(containerDef.CPU)
	}

	// Port mappings
	config.PortMap = make(map[string]string)
	for _, port := range containerDef.PortMappings {
		containerPort := strconv.Itoa(port.ContainerPort)
		hostPort := strconv.Itoa(port.GetEffectiveHostPort())
		config.PortMap[containerPort] = hostPort
	}

	return config
}

// generateTaskID generates a unique task ID
func generateTaskID(family string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", strings.ToLower(family), timestamp)
}

// cleanupFailedTask cleans up resources when task creation fails
func (m *Manager) cleanupFailedTask(ctx context.Context, task *Task) {
	for name, containerID := range task.Containers {
		err := m.dockerClient.RemoveContainer(ctx, containerID, true)
		if err != nil {
			fmt.Printf("Warning: failed to cleanup container %s: %v\n", name, err)
		}
	}

	if task.NetworkID != "" {
		err := m.dockerClient.RemoveNetwork(ctx, task.NetworkID)
		if err != nil {
			fmt.Printf("Warning: failed to cleanup network %s: %v\n", task.NetworkName, err)
		}
	}
}
