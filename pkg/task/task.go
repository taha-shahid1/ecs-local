package task

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
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
	Volumes     []string
	Status      string
	StartedAt   time.Time
	Definition  *parser.TaskDefinition
}

// Manager handles task lifecycle
type Manager struct {
	dockerClient    *docker.Client
	tasks           map[string]*Task // taskID -> Task
	containerStates map[string]*ContainerState // containerID -> state
	statesMu        sync.RWMutex
	cascadeOnFailure bool // Whether to stop dependent containers when dependency fails
}

// NewManager initializes a task manager
func NewManager(dockerClient *docker.Client) *Manager {
	return &Manager{
		dockerClient:     dockerClient,
		tasks:            make(map[string]*Task),
		containerStates:  make(map[string]*ContainerState),
		cascadeOnFailure: true, // Default to cascade failure
	}
}

// SetCascadeOnFailure configures whether to stop dependent containers when a dependency fails
func (m *Manager) SetCascadeOnFailure(cascade bool) {
	m.cascadeOnFailure = cascade
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

	// Create named volumes
	volumeMap := make(map[string]string)
	for _, vol := range taskDef.Volumes {
		if vol.Host == nil || vol.Host.SourcePath == "" {
			volumeName := fmt.Sprintf("ecs-local-%s-%s", taskID, vol.Name)
			_, err := m.dockerClient.CreateVolume(ctx, volumeName)
			if err != nil {
				task.Status = "FAILED"
				return nil, fmt.Errorf("failed to create volume %s: %w", vol.Name, err)
			}
			task.Volumes = append(task.Volumes, volumeName)
			volumeMap[vol.Name] = volumeName
			fmt.Printf("Created volume: %s\n", volumeName)
		} else {
			volumeMap[vol.Name] = vol.Host.SourcePath
		}
	}

	// Pull all images first
	for _, containerDef := range taskDef.ContainerDefinitions {
		fmt.Printf("Pulling image: %s\n", containerDef.Image)
		err := m.dockerClient.PullImage(ctx, containerDef.Image, true)
		if err != nil {
			task.Status = "FAILED"
			return nil, fmt.Errorf("failed to pull image %s: %w", containerDef.Image, err)
		}
	}

	// Build dependency graph
	log.Printf("Building dependency graph for task %s", taskID)
	depGraph, err := buildDependencyGraph(taskDef.ContainerDefinitions)
	if err != nil {
		task.Status = "FAILED"
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}
	
	// Log dependency graph structure
	log.Printf("Dependency graph built successfully:")
	for name, deps := range depGraph.deps {
		if len(deps) > 0 {
			depNames := make([]string, len(deps))
			for i, dep := range deps {
				depNames[i] = fmt.Sprintf("%s(%s)", dep.ContainerName, dep.Condition)
			}
			log.Printf("  %s depends on: %v", name, depNames)
		} else {
			log.Printf("  %s has no dependencies", name)
		}
	}

	// Create all containers first (don't start yet)
	for _, containerDef := range taskDef.ContainerDefinitions {
		containerConfig := convertToDockerConfig(taskID, containerDef, volumeMap)

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

	// Initialize container state tracking
	for _, containerID := range task.Containers {
		m.statesMu.Lock()
		m.containerStates[containerID] = &ContainerState{
			Status:    "created",
			ExitCode:  -1,
			Health:    "none",
			LastCheck: time.Now(),
		}
		m.statesMu.Unlock()
	}

	// Start containers in dependency levels (parallel execution within levels)
	startLevels := depGraph.getStartLevels()
	log.Printf("Starting %d containers in %d dependency levels", len(task.Containers), len(startLevels))

	for levelIdx, level := range startLevels {
		log.Printf("Starting level %d: %v", levelIdx+1, level)
		
		// Start all containers in this level in parallel
		var wg sync.WaitGroup
		startErrors := make(chan error, len(level))
		
		for _, name := range level {
			wg.Add(1)
			go func(containerName string) {
				defer wg.Done()
				
				containerID := task.Containers[containerName]
				containerDef := depGraph.containers[containerName]

				// Wait for all dependencies to satisfy their conditions
				for _, dep := range containerDef.DependsOn {
					depContainerID := task.Containers[dep.ContainerName]
					log.Printf("[%s] Waiting for dependency %s (condition: %s)...", containerName, dep.ContainerName, dep.Condition)
					fmt.Printf("[%s] Waiting for %s (condition: %s)...\n", containerName, dep.ContainerName, dep.Condition)
					
					err := m.waitForCondition(ctx, depContainerID, dep.Condition, containerName)
					if err != nil {
						log.Printf("[%s] Dependency %s failed: %v", containerName, dep.ContainerName, err)
						startErrors <- fmt.Errorf("dependency %s failed for container %s: %w", dep.ContainerName, containerName, err)
						
						// Handle cascade failure if enabled
						if m.cascadeOnFailure {
							m.handleDependencyFailure(ctx, task, containerName, depGraph)
						}
						return
					}
					log.Printf("[%s] Dependency %s satisfied condition %s", containerName, dep.ContainerName, dep.Condition)
				}

				// Start the container
				err := m.dockerClient.StartContainer(ctx, containerID)
				if err != nil {
					log.Printf("[%s] Failed to start: %v", containerName, err)
					startErrors <- fmt.Errorf("failed to start container %s: %w", containerName, err)
					return
				}
				
				// Update state
				m.updateContainerState(containerID, "running", -1, "none")
				log.Printf("[%s] Started successfully", containerName)
				fmt.Printf("Started container: %s\n", containerName)
			}(name)
		}

		wg.Wait()
		close(startErrors)

		// Check for errors
		var errors []error
		for err := range startErrors {
			errors = append(errors, err)
		}

		if len(errors) > 0 {
			task.Status = "FAILED"
			return nil, fmt.Errorf("failed to start containers in level %d: %v", levelIdx+1, errors[0])
		}
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

	for _, volumeName := range task.Volumes {
		err := m.dockerClient.RemoveVolume(ctx, volumeName, false)
		if err != nil {
			fmt.Printf("Warning: failed to remove volume %s: %v\n", volumeName, err)
		} else {
			fmt.Printf("Removed volume: %s\n", volumeName)
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

// GetTaskDependencyInfo returns dependency information for a task
func (m *Manager) GetTaskDependencyInfo(taskID string) (*DependencyInfo, error) {
	task, err := m.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	depGraph, err := buildDependencyGraph(task.Definition.ContainerDefinitions)
	if err != nil {
		return nil, err
	}

	info := &DependencyInfo{
		TaskID:      taskID,
		Containers:  make(map[string]ContainerDependencyInfo),
		StartLevels: depGraph.getStartLevels(),
	}

	ctx := context.Background()
	for name, containerID := range task.Containers {
		containerInfo := ContainerDependencyInfo{
			ContainerID: containerID,
			Dependencies: depGraph.GetDependencies(name),
			Dependents:   depGraph.GetDependents(name),
		}

		// Get current state
		state := m.getContainerState(containerID)
		if state != nil {
			state.mu.RLock()
			containerInfo.Status = state.Status
			containerInfo.ExitCode = state.ExitCode
			containerInfo.Health = state.Health
			state.mu.RUnlock()
		} else {
			// Fallback to Docker status
			status, _ := m.dockerClient.GetContainerStatus(ctx, containerID)
			containerInfo.Status = status
		}

		// Check if waiting for dependencies
		if containerInfo.Status != "running" && containerInfo.Status != "exited" {
			for _, dep := range containerInfo.Dependencies {
				depContainerID := task.Containers[dep.ContainerName]
				depState := m.getContainerState(depContainerID)
				if depState != nil {
					depState.mu.RLock()
					depStatus := depState.Status
					depExitCode := depState.ExitCode
					depHealth := depState.Health
					depState.mu.RUnlock()

					if !m.isConditionSatisfied(depStatus, depExitCode, depHealth, dep.Condition) {
						containerInfo.WaitingFor = append(containerInfo.WaitingFor, WaitingDependency{
							ContainerName: dep.ContainerName,
							Condition:     dep.Condition,
							CurrentStatus: depStatus,
							CurrentHealth: depHealth,
							ExitCode:      depExitCode,
						})
					}
				}
			}
		}

		info.Containers[name] = containerInfo
	}

	return info, nil
}

// isConditionSatisfied checks if a dependency condition is satisfied
func (m *Manager) isConditionSatisfied(status string, exitCode int, health string, condition string) bool {
	switch condition {
	case "START":
		return status == "running"
	case "COMPLETE":
		return status == "exited"
	case "SUCCESS":
		return status == "exited" && exitCode == 0
	case "HEALTHY":
		return health == "healthy"
	default:
		return false
	}
}

// DependencyInfo contains dependency information for a task
type DependencyInfo struct {
	TaskID      string
	Containers  map[string]ContainerDependencyInfo
	StartLevels [][]string
}

// ContainerDependencyInfo contains dependency information for a container
type ContainerDependencyInfo struct {
	ContainerID  string
	Status       string
	ExitCode     int
	Health       string
	Dependencies []DependencyCondition
	Dependents   []string
	WaitingFor   []WaitingDependency
}

// WaitingDependency represents a dependency that a container is waiting for
type WaitingDependency struct {
	ContainerName string
	Condition     string
	CurrentStatus string
	CurrentHealth string
	ExitCode      int
}

// FormatDependencyGraph returns a string representation of the dependency graph
func (info *DependencyInfo) FormatDependencyGraph() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dependency Graph for Task: %s\n", info.TaskID))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Show start levels
	sb.WriteString("Start Levels (containers at same level start in parallel):\n")
	for i, level := range info.StartLevels {
		sb.WriteString(fmt.Sprintf("  Level %d: %v\n", i+1, level))
	}
	sb.WriteString("\n")

	// Show dependencies for each container
	sb.WriteString("Container Dependencies:\n")
	for name, containerInfo := range info.Containers {
		sb.WriteString(fmt.Sprintf("  %s [%s", name, containerInfo.Status))
		if containerInfo.Health != "none" {
			sb.WriteString(fmt.Sprintf(", health: %s", containerInfo.Health))
		}
		if containerInfo.ExitCode >= 0 {
			sb.WriteString(fmt.Sprintf(", exit: %d", containerInfo.ExitCode))
		}
		sb.WriteString("]\n")

		if len(containerInfo.Dependencies) > 0 {
			sb.WriteString("    Depends on:\n")
			for _, dep := range containerInfo.Dependencies {
				sb.WriteString(fmt.Sprintf("      - %s (%s)\n", dep.ContainerName, dep.Condition))
			}
		}

		if len(containerInfo.WaitingFor) > 0 {
			sb.WriteString("    ⏳ Waiting for:\n")
			for _, waiting := range containerInfo.WaitingFor {
				sb.WriteString(fmt.Sprintf("      - %s (%s) [current: %s", waiting.ContainerName, waiting.Condition, waiting.CurrentStatus))
				if waiting.CurrentHealth != "none" {
					sb.WriteString(fmt.Sprintf(", health: %s", waiting.CurrentHealth))
				}
				if waiting.ExitCode >= 0 {
					sb.WriteString(fmt.Sprintf(", exit: %d", waiting.ExitCode))
				}
				sb.WriteString("]\n")
			}
		}

		if len(containerInfo.Dependents) > 0 {
			sb.WriteString("    Dependents:\n")
			for _, dependent := range containerInfo.Dependents {
				sb.WriteString(fmt.Sprintf("      - %s\n", dependent))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// convertToDockerConfig converts a parser.ContainerDefinition to docker.ContainerConfig
func convertToDockerConfig(taskID string, containerDef parser.ContainerDefinition, volumeMap map[string]string) docker.ContainerConfig {
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

	// Health check
	if containerDef.HealthCheck != nil {
		hc := containerDef.HealthCheck
		config.HealthCheck = &docker.HealthCheckConfig{
			Test:        hc.Command,
			Interval:    int64(hc.Interval) * 1000000000,
			Timeout:     int64(hc.Timeout) * 1000000000,
			Retries:     hc.Retries,
			StartPeriod: int64(hc.StartPeriod) * 1000000000,
		}

		if config.HealthCheck.Interval == 0 {
			config.HealthCheck.Interval = 30 * 1000000000
		}
		if config.HealthCheck.Timeout == 0 {
			config.HealthCheck.Timeout = 5 * 1000000000
		}
		if config.HealthCheck.Retries == 0 {
			config.HealthCheck.Retries = 3
		}
	}

	// Mount points
	for _, mountPoint := range containerDef.MountPoints {
		source, exists := volumeMap[mountPoint.SourceVolume]
		if !exists {
			continue
		}

		mountType := "volume"
		if source[0] == '/' || source[0] == '.' {
			mountType = "bind"
		}

		config.Mounts = append(config.Mounts, docker.MountConfig{
			Source:   source,
			Target:   mountPoint.ContainerPath,
			ReadOnly: mountPoint.ReadOnly,
			Type:     mountType,
		})
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

	for _, volumeName := range task.Volumes {
		err := m.dockerClient.RemoveVolume(ctx, volumeName, true)
		if err != nil {
			fmt.Printf("Warning: failed to cleanup volume %s: %v\n", volumeName, err)
		}
	}
}
