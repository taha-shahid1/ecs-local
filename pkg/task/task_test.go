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
