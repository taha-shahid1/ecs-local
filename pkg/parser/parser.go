package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

// TaskDefinition represents an ECS task definition
type TaskDefinition struct {
	Family                  string                `json:"family"`
	TaskRoleArn             string                `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn        string                `json:"executionRoleArn,omitempty"`
	NetworkMode             string                `json:"networkMode,omitempty"`
	ContainerDefinitions    []ContainerDefinition `json:"containerDefinitions"`
	Volumes                 []Volume              `json:"volumes,omitempty"`
	RequiresCompatibilities []string              `json:"requiresCompatibilities,omitempty"`
	CPU                     string                `json:"cpu,omitempty"`
	Memory                  string                `json:"memory,omitempty"`
}

// ContainerDefinition represents a container in an ECS task
type ContainerDefinition struct {
	Name              string                 `json:"name"`
	Image             string                 `json:"image"`
	CPU               int                    `json:"cpu,omitempty"`
	Memory            int                    `json:"memory,omitempty"`
	MemoryReservation int                    `json:"memoryReservation,omitempty"`
	Essential         bool                   `json:"essential,omitempty"`
	EntryPoint        []string               `json:"entryPoint,omitempty"`
	Command           []string               `json:"command,omitempty"`
	Environment       []EnvironmentVariable  `json:"environment,omitempty"`
	PortMappings      []PortMapping          `json:"portMappings,omitempty"`
	DependsOn         []ContainerDependency  `json:"dependsOn,omitempty"`
	Links             []string               `json:"links,omitempty"`
	HealthCheck       *HealthCheck           `json:"healthCheck,omitempty"`
	LogConfiguration  *LogConfiguration      `json:"logConfiguration,omitempty"`
	MountPoints       []MountPoint           `json:"mountPoints,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PortMapping represents a port mapping
type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

// ContainerDependency represents a container dependency
type ContainerDependency struct {
	ContainerName string `json:"containerName"`
	Condition     string `json:"condition"`
}

// HealthCheck represents container health check configuration
type HealthCheck struct {
	Command     []string `json:"command"`
	Interval    int      `json:"interval,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
	Retries     int      `json:"retries,omitempty"`
	StartPeriod int      `json:"startPeriod,omitempty"`
}

// LogConfiguration represents logging configuration
type LogConfiguration struct {
	LogDriver string            `json:"logDriver"`
	Options   map[string]string `json:"options,omitempty"`
}

// Volume represents a volume definition
type Volume struct {
	Name string `json:"name"`
	Host *Host  `json:"host,omitempty"`
}

// Host represents a host volume
type Host struct {
	SourcePath string `json:"sourcePath,omitempty"`
}

// MountPoint represents a volume mount point
type MountPoint struct {
	SourceVolume  string `json:"sourceVolume"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
}

// ParseTaskDefinition parses and validates a task definition from a JSON file
func ParseTaskDefinition(filePath string) (*TaskDefinition, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task definition file: %w", err)
	}

	var taskDef TaskDefinition
	if err := json.Unmarshal(data, &taskDef); err != nil {
		return nil, fmt.Errorf("failed to parse task definition JSON: %w", err)
	}

	if err := taskDef.Validate(); err != nil {
		return nil, fmt.Errorf("task definition validation failed: %w", err)
	}

	return &taskDef, nil
}

// ParseTaskDefinitionFromJSON parses and validates a task definition from JSON bytes
func ParseTaskDefinitionFromJSON(data []byte) (*TaskDefinition, error) {
	var taskDef TaskDefinition
	if err := json.Unmarshal(data, &taskDef); err != nil {
		return nil, fmt.Errorf("failed to parse task definition JSON: %w", err)
	}

	if err := taskDef.Validate(); err != nil {
		return nil, fmt.Errorf("task definition validation failed: %w", err)
	}

	return &taskDef, nil
}

// Validate validates required fields in the task definition
func (td *TaskDefinition) Validate() error {
	if td.Family == "" {
		return fmt.Errorf("family is required")
	}

	if len(td.ContainerDefinitions) == 0 {
		return fmt.Errorf("at least one container definition is required")
	}

	for i, container := range td.ContainerDefinitions {
		if err := container.Validate(); err != nil {
			return fmt.Errorf("container definition [%d] validation failed: %w", i, err)
		}
	}

	return nil
}

// Validate validates required fields in the container definition
func (cd *ContainerDefinition) Validate() error {
	if cd.Name == "" {
		return fmt.Errorf("container name is required")
	}

	if cd.Image == "" {
		return fmt.Errorf("container image is required for container '%s'", cd.Name)
	}

	for i, port := range cd.PortMappings {
		if err := port.Validate(); err != nil {
			return fmt.Errorf("port mapping [%d] validation failed for container '%s': %w", i, cd.Name, err)
		}
	}

	for i, dep := range cd.DependsOn {
		if err := dep.Validate(); err != nil {
			return fmt.Errorf("dependency [%d] validation failed for container '%s': %w", i, cd.Name, err)
		}
	}

	return nil
}

// Validate validates the port mapping configuration
func (pm *PortMapping) Validate() error {
	if pm.ContainerPort <= 0 {
		return fmt.Errorf("containerPort must be greater than 0")
	}

	if pm.ContainerPort > 65535 {
		return fmt.Errorf("containerPort must be less than or equal to 65535")
	}

	if pm.HostPort < 0 || pm.HostPort > 65535 {
		return fmt.Errorf("hostPort must be between 0 and 65535")
	}

	if pm.Protocol == "" {
		pm.Protocol = "tcp"
	}

	if pm.Protocol != "tcp" && pm.Protocol != "udp" {
		return fmt.Errorf("protocol must be 'tcp' or 'udp'")
	}

	return nil
}

// Validate validates the container dependency configuration
func (cd *ContainerDependency) Validate() error {
	if cd.ContainerName == "" {
		return fmt.Errorf("containerName is required for dependency")
	}

	validConditions := map[string]bool{
		"START":    true,
		"COMPLETE": true,
		"SUCCESS":  true,
		"HEALTHY":  true,
	}

	if cd.Condition == "" {
		cd.Condition = "START"
	}

	if !validConditions[cd.Condition] {
		return fmt.Errorf("invalid dependency condition '%s', must be one of: START, COMPLETE, SUCCESS, HEALTHY", cd.Condition)
	}

	return nil
}

// GetEffectiveMemory returns Memory if set, otherwise MemoryReservation
func (cd *ContainerDefinition) GetEffectiveMemory() int {
	if cd.Memory > 0 {
		return cd.Memory
	}
	return cd.MemoryReservation
}

// GetEffectiveHostPort returns HostPort if set, otherwise ContainerPort
func (pm *PortMapping) GetEffectiveHostPort() int {
	if pm.HostPort > 0 {
		return pm.HostPort
	}
	return pm.ContainerPort
}
