package task

import (
	"context"
	"testing"

	"github.com/taha-shahid1/ecs-local/pkg/docker"
	"github.com/taha-shahid1/ecs-local/pkg/parser"
)

func TestNewManager(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	if manager == nil {
		t.Error("expected non-nil manager")
	}

	if manager.dockerClient == nil {
		t.Error("expected docker client to be set")
	}

	if manager.tasks == nil {
		t.Error("expected tasks map to be initialized")
	}
}

func TestManager_RunTask(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "test-task",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:   "test-container",
				Image:  "alpine:latest",
				Memory: 128,
				CPU:    256,
				Command: []string{"sleep", "5"},
			},
		},
	}

	task, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Errorf("failed to run task: %v", err)
		return
	}
	defer manager.RemoveTask(ctx, task.ID)

	if task.ID == "" {
		t.Error("expected non-empty task ID")
	}

	if task.Family != "test-task" {
		t.Errorf("expected family 'test-task', got '%s'", task.Family)
	}

	if len(task.Containers) != 1 {
		t.Errorf("expected 1 container, got %d", len(task.Containers))
	}

	if task.Status != "RUNNING" {
		t.Errorf("expected status RUNNING, got %s", task.Status)
	}

	t.Logf("Task created: %s", task.ID)
}

func TestManager_RunTask_MultiContainer(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "multi-container-task",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:   "web",
				Image:  "nginx:alpine",
				Memory: 256,
				PortMappings: []parser.PortMapping{
					{ContainerPort: 80, HostPort: 8081},
				},
			},
			{
				Name:   "app",
				Image:  "alpine:latest",
				Memory: 128,
				Command: []string{"sleep", "5"},
			},
		},
	}

	task, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Errorf("failed to run multi-container task: %v", err)
		return
	}
	defer manager.RemoveTask(ctx, task.ID)

	if len(task.Containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(task.Containers))
	}

	if _, exists := task.Containers["web"]; !exists {
		t.Error("expected 'web' container")
	}

	if _, exists := task.Containers["app"]; !exists {
		t.Error("expected 'app' container")
	}

	t.Logf("Multi-container task created: %s", task.ID)
}

func TestManager_RunTask_WithEnv(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "task-with-env",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:   "app",
				Image:  "alpine:latest",
				Memory: 128,
				Environment: []parser.EnvironmentVariable{
					{Name: "TEST_VAR", Value: "test_value"},
					{Name: "ANOTHER", Value: "value"},
				},
				Command: []string{"env"},
			},
		},
	}

	task, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Errorf("failed to run task with env: %v", err)
		return
	}
	defer manager.RemoveTask(ctx, task.ID)

	t.Logf("Task with env vars created: %s", task.ID)
}

func TestManager_StopTask(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "task-to-stop",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:    "app",
				Image:   "alpine:latest",
				Memory:  128,
				Command: []string{"sleep", "30"},
			},
		},
	}

	task, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Skipf("failed to run task: %v", err)
	}
	defer manager.RemoveTask(ctx, task.ID)

	err = manager.StopTask(ctx, task.ID)
	if err != nil {
		t.Errorf("failed to stop task: %v", err)
	}

	if task.Status != "STOPPED" {
		t.Errorf("expected status STOPPED, got %s", task.Status)
	}
}

func TestManager_GetTask(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "test-get-task",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:    "app",
				Image:   "alpine:latest",
				Memory:  128,
				Command: []string{"sleep", "5"},
			},
		},
	}

	createdTask, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Skipf("failed to run task: %v", err)
	}
	defer manager.RemoveTask(ctx, createdTask.ID)

	retrievedTask, err := manager.GetTask(createdTask.ID)
	if err != nil {
		t.Errorf("failed to get task: %v", err)
	}

	if retrievedTask.ID != createdTask.ID {
		t.Errorf("expected task ID %s, got %s", createdTask.ID, retrievedTask.ID)
	}
}

func TestManager_GetTask_NotFound(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)

	_, err = manager.GetTask("nonexistent-task")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestManager_ListTasks(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "test-list-task",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:    "app",
				Image:   "alpine:latest",
				Memory:  128,
				Command: []string{"sleep", "5"},
			},
		},
	}

	task, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Skipf("failed to run task: %v", err)
	}
	defer manager.RemoveTask(ctx, task.ID)

	tasks := manager.ListTasks()
	if len(tasks) < 1 {
		t.Error("expected at least 1 task")
	}

	found := false
	for _, t := range tasks {
		if t.ID == task.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("created task not found in list")
	}
}

func TestConvertToDockerConfig(t *testing.T) {
	containerDef := parser.ContainerDefinition{
		Name:   "test",
		Image:  "nginx:alpine",
		Memory: 512,
		CPU:    256,
		Environment: []parser.EnvironmentVariable{
			{Name: "KEY", Value: "value"},
		},
		PortMappings: []parser.PortMapping{
			{ContainerPort: 80, HostPort: 8080},
		},
		Command:    []string{"nginx", "-g", "daemon off;"},
		EntryPoint: []string{"/bin/sh"},
	}

	config := convertToDockerConfig("task-123", containerDef, make(map[string]string))

	if config.Image != "nginx:alpine" {
		t.Errorf("expected image nginx:alpine, got %s", config.Image)
	}

	if config.Memory != 512*1024*1024 {
		t.Errorf("expected memory 536870912, got %d", config.Memory)
	}

	if config.CPU != 256 {
		t.Errorf("expected CPU 256, got %d", config.CPU)
	}

	if len(config.Env) != 1 || config.Env[0] != "KEY=value" {
		t.Errorf("unexpected env vars: %v", config.Env)
	}

	if len(config.PortMap) != 1 {
		t.Errorf("expected 1 port mapping, got %d", len(config.PortMap))
	}

	if config.PortMap["80"] != "8080" {
		t.Errorf("expected port mapping 80:8080, got %v", config.PortMap)
	}

	if len(config.Cmd) != 3 {
		t.Errorf("expected 3 command args, got %d", len(config.Cmd))
	}

	if len(config.Entrypoint) != 1 {
		t.Errorf("expected 1 entrypoint arg, got %d", len(config.Entrypoint))
	}
}

func TestManager_SetCascadeOnFailure(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)

	// Test default value
	if !manager.cascadeOnFailure {
		t.Error("expected cascadeOnFailure to be true by default")
	}

	// Test setting to false
	manager.SetCascadeOnFailure(false)
	if manager.cascadeOnFailure {
		t.Error("expected cascadeOnFailure to be false after SetCascadeOnFailure(false)")
	}

	// Test setting back to true
	manager.SetCascadeOnFailure(true)
	if !manager.cascadeOnFailure {
		t.Error("expected cascadeOnFailure to be true after SetCascadeOnFailure(true)")
	}
}

func TestManager_isConditionSatisfied(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)

	// Test START condition
	if !manager.isConditionSatisfied("running", -1, "none", "START") {
		t.Error("expected START condition to be satisfied when status is running")
	}
	if manager.isConditionSatisfied("created", -1, "none", "START") {
		t.Error("expected START condition to not be satisfied when status is created")
	}
	if manager.isConditionSatisfied("exited", -1, "none", "START") {
		t.Error("expected START condition to not be satisfied when status is exited")
	}

	// Test COMPLETE condition
	if !manager.isConditionSatisfied("exited", -1, "none", "COMPLETE") {
		t.Error("expected COMPLETE condition to be satisfied when status is exited")
	}
	if manager.isConditionSatisfied("running", -1, "none", "COMPLETE") {
		t.Error("expected COMPLETE condition to not be satisfied when status is running")
	}

	// Test SUCCESS condition
	if !manager.isConditionSatisfied("exited", 0, "none", "SUCCESS") {
		t.Error("expected SUCCESS condition to be satisfied when status is exited and exitCode is 0")
	}
	if manager.isConditionSatisfied("exited", 1, "none", "SUCCESS") {
		t.Error("expected SUCCESS condition to not be satisfied when exitCode is non-zero")
	}
	if manager.isConditionSatisfied("running", -1, "none", "SUCCESS") {
		t.Error("expected SUCCESS condition to not be satisfied when status is not exited")
	}

	// Test HEALTHY condition
	if !manager.isConditionSatisfied("running", -1, "healthy", "HEALTHY") {
		t.Error("expected HEALTHY condition to be satisfied when health is healthy")
	}
	if manager.isConditionSatisfied("running", -1, "unhealthy", "HEALTHY") {
		t.Error("expected HEALTHY condition to not be satisfied when health is unhealthy")
	}
	if manager.isConditionSatisfied("running", -1, "none", "HEALTHY") {
		t.Error("expected HEALTHY condition to not be satisfied when health is none")
	}

	// Test unknown condition
	if manager.isConditionSatisfied("running", -1, "none", "UNKNOWN") {
		t.Error("expected unknown condition to not be satisfied")
	}
}

func TestManager_updateContainerState(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)

	containerID := "test-container-id"

	// Test initial state update
	manager.updateContainerState(containerID, "running", -1, "none")
	state := manager.getContainerState(containerID)
	if state == nil {
		t.Fatal("expected state to be created")
	}

	state.mu.RLock()
	if state.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", state.Status)
	}
	if state.Health != "none" {
		t.Errorf("expected health 'none', got '%s'", state.Health)
	}
	state.mu.RUnlock()

	// Test updating with exit code
	manager.updateContainerState(containerID, "exited", 0, "none")
	state = manager.getContainerState(containerID)
	state.mu.RLock()
	if state.Status != "exited" {
		t.Errorf("expected status 'exited', got '%s'", state.Status)
	}
	if state.ExitCode != 0 {
		t.Errorf("expected exitCode 0, got %d", state.ExitCode)
	}
	state.mu.RUnlock()

	// Test updating with health
	manager.updateContainerState(containerID, "running", -1, "healthy")
	state = manager.getContainerState(containerID)
	state.mu.RLock()
	if state.Health != "healthy" {
		t.Errorf("expected health 'healthy', got '%s'", state.Health)
	}
	state.mu.RUnlock()

	// Test that exit code is not overwritten if not provided
	manager.updateContainerState(containerID, "running", -1, "healthy")
	state = manager.getContainerState(containerID)
	state.mu.RLock()
	if state.ExitCode != 0 {
		t.Errorf("expected exitCode to remain 0, got %d", state.ExitCode)
	}
	state.mu.RUnlock()
}

func TestManager_getContainerState(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)

	// Test getting non-existent state
	state := manager.getContainerState("nonexistent")
	if state != nil {
		t.Error("expected nil state for non-existent container")
	}

	// Test getting existing state
	containerID := "test-container"
	manager.updateContainerState(containerID, "running", -1, "none")
	state = manager.getContainerState(containerID)
	if state == nil {
		t.Fatal("expected non-nil state")
	}

	state.mu.RLock()
	if state.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", state.Status)
	}
	state.mu.RUnlock()
}

func TestManager_RunTask_WithDependencies(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "task-with-deps",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:    "init",
				Image:   "alpine:latest",
				Memory:  128,
				Command: []string{"sleep", "2"},
			},
			{
				Name:    "app",
				Image:   "alpine:latest",
				Memory:  128,
				Command: []string{"sleep", "5"},
				DependsOn: []parser.ContainerDependency{
					{ContainerName: "init", Condition: "START"},
				},
			},
		},
	}

	task, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Errorf("failed to run task with dependencies: %v", err)
		return
	}
	defer manager.RemoveTask(ctx, task.ID)

	if len(task.Containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(task.Containers))
	}

	// Verify both containers exist
	if _, exists := task.Containers["init"]; !exists {
		t.Error("expected 'init' container")
	}
	if _, exists := task.Containers["app"]; !exists {
		t.Error("expected 'app' container")
	}

	// Verify state tracking was initialized
	initState := manager.getContainerState(task.Containers["init"])
	if initState == nil {
		t.Error("expected state to be tracked for init container")
	}

	appState := manager.getContainerState(task.Containers["app"])
	if appState == nil {
		t.Error("expected state to be tracked for app container")
	}
}

func TestManager_GetTaskDependencyInfo(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer dockerClient.Close()

	manager := NewManager(dockerClient)
	ctx := context.Background()

	taskDef := &parser.TaskDefinition{
		Family: "task-for-deps-info",
		ContainerDefinitions: []parser.ContainerDefinition{
			{
				Name:    "db",
				Image:   "alpine:latest",
				Memory:  128,
				Command: []string{"sleep", "5"},
			},
			{
				Name:    "app",
				Image:   "alpine:latest",
				Memory:  128,
				Command: []string{"sleep", "5"},
				DependsOn: []parser.ContainerDependency{
					{ContainerName: "db", Condition: "START"},
				},
			},
		},
	}

	task, err := manager.RunTask(ctx, taskDef)
	if err != nil {
		t.Skipf("failed to run task: %v", err)
	}
	defer manager.RemoveTask(ctx, task.ID)

	info, err := manager.GetTaskDependencyInfo(task.ID)
	if err != nil {
		t.Errorf("failed to get dependency info: %v", err)
		return
	}

	if info.TaskID != task.ID {
		t.Errorf("expected taskID %s, got %s", task.ID, info.TaskID)
	}

	if len(info.Containers) != 2 {
		t.Errorf("expected 2 containers in info, got %d", len(info.Containers))
	}

	// Check db container info
	dbInfo, exists := info.Containers["db"]
	if !exists {
		t.Fatal("expected db container in info")
	}

	if len(dbInfo.Dependencies) != 0 {
		t.Errorf("expected db to have 0 dependencies, got %d", len(dbInfo.Dependencies))
	}

	if len(dbInfo.Dependents) != 1 || dbInfo.Dependents[0] != "app" {
		t.Errorf("expected db to have 1 dependent 'app', got %v", dbInfo.Dependents)
	}

	// Check app container info
	appInfo, exists := info.Containers["app"]
	if !exists {
		t.Fatal("expected app container in info")
	}

	if len(appInfo.Dependencies) != 1 {
		t.Errorf("expected app to have 1 dependency, got %d", len(appInfo.Dependencies))
	}

	if appInfo.Dependencies[0].ContainerName != "db" || appInfo.Dependencies[0].Condition != "START" {
		t.Errorf("expected app to depend on db with START condition, got %v", appInfo.Dependencies[0])
	}

	// Check start levels
	if len(info.StartLevels) < 2 {
		t.Errorf("expected at least 2 start levels, got %d", len(info.StartLevels))
	}

	// Level 1 should have db
	foundDB := false
	for _, name := range info.StartLevels[0] {
		if name == "db" {
			foundDB = true
			break
		}
	}
	if !foundDB {
		t.Errorf("expected db to be in first start level, got %v", info.StartLevels[0])
	}
}

func TestDependencyInfo_FormatDependencyGraph(t *testing.T) {
	info := &DependencyInfo{
		TaskID: "test-task-123",
		Containers: map[string]ContainerDependencyInfo{
			"db": {
				ContainerID:  "db-id",
				Status:       "running",
				ExitCode:     -1,
				Health:       "none",
				Dependencies: []DependencyCondition{},
				Dependents:   []string{"app"},
				WaitingFor:   []WaitingDependency{},
			},
			"app": {
				ContainerID: "app-id",
				Status:     "running",
				ExitCode:   -1,
				Health:     "none",
				Dependencies: []DependencyCondition{
					{ContainerName: "db", Condition: "START"},
				},
				Dependents: []string{},
				WaitingFor: []WaitingDependency{},
			},
		},
		StartLevels: [][]string{
			{"db"},
			{"app"},
		},
	}

	graphStr := info.FormatDependencyGraph()

	if graphStr == "" {
		t.Error("expected non-empty dependency graph string")
	}

	// Check that key information is present
	if !contains(graphStr, "test-task-123") {
		t.Error("expected task ID in graph string")
	}

	if !contains(graphStr, "db") {
		t.Error("expected db container in graph string")
	}

	if !contains(graphStr, "app") {
		t.Error("expected app container in graph string")
	}

	if !contains(graphStr, "START") {
		t.Error("expected START condition in graph string")
	}

	if !contains(graphStr, "Level 1") {
		t.Error("expected start levels in graph string")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
