package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTaskDefinitionFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal task definition",
			json: `{
				"family": "my-app",
				"containerDefinitions": [
					{
						"name": "web",
						"image": "nginx:latest",
						"memory": 512
					}
				]
			}`,
			wantErr: false,
		},
		{
			name: "valid task with multiple containers",
			json: `{
				"family": "multi-container-app",
				"containerDefinitions": [
					{
						"name": "web",
						"image": "nginx:latest",
						"memory": 512,
						"cpu": 256,
						"essential": true,
						"portMappings": [
							{
								"containerPort": 80,
								"hostPort": 8080,
								"protocol": "tcp"
							}
						],
						"environment": [
							{
								"name": "ENV",
								"value": "production"
							}
						]
					},
					{
						"name": "app",
						"image": "myapp:v1.0",
						"memory": 1024,
						"dependsOn": [
							{
								"containerName": "web",
								"condition": "HEALTHY"
							}
						]
					}
				]
			}`,
			wantErr: false,
		},
		{
			name: "missing family",
			json: `{
				"containerDefinitions": [
					{
						"name": "web",
						"image": "nginx:latest"
					}
				]
			}`,
			wantErr: true,
			errMsg:  "family is required",
		},
		{
			name: "missing container definitions",
			json: `{
				"family": "my-app",
				"containerDefinitions": []
			}`,
			wantErr: true,
			errMsg:  "at least one container definition is required",
		},
		{
			name: "missing container name",
			json: `{
				"family": "my-app",
				"containerDefinitions": [
					{
						"image": "nginx:latest"
					}
				]
			}`,
			wantErr: true,
			errMsg:  "container name is required",
		},
		{
			name: "missing container image",
			json: `{
				"family": "my-app",
				"containerDefinitions": [
					{
						"name": "web"
					}
				]
			}`,
			wantErr: true,
			errMsg:  "container image is required",
		},
		{
			name: "invalid port mapping - port too high",
			json: `{
				"family": "my-app",
				"containerDefinitions": [
					{
						"name": "web",
						"image": "nginx:latest",
						"portMappings": [
							{
								"containerPort": 99999
							}
						]
					}
				]
			}`,
			wantErr: true,
			errMsg:  "containerPort must be less than or equal to 65535",
		},
		{
			name: "invalid dependency condition",
			json: `{
				"family": "my-app",
				"containerDefinitions": [
					{
						"name": "web",
						"image": "nginx:latest",
						"dependsOn": [
							{
								"containerName": "db",
								"condition": "INVALID"
							}
						]
					}
				]
			}`,
			wantErr: true,
			errMsg:  "invalid dependency condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskDef, err := ParseTaskDefinitionFromJSON([]byte(tt.json))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if taskDef == nil {
				t.Error("expected task definition but got nil")
			}
		})
	}
}

func TestParseTaskDefinitionFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "parser-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid task definition file
	validTaskDef := `{
		"family": "test-app",
		"containerDefinitions": [
			{
				"name": "web",
				"image": "nginx:latest",
				"memory": 512,
				"portMappings": [
					{
						"containerPort": 80,
						"hostPort": 8080
					}
				]
			}
		]
	}`

	validFilePath := filepath.Join(tmpDir, "valid-task.json")
	if err := os.WriteFile(validFilePath, []byte(validTaskDef), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test valid file
	taskDef, err := ParseTaskDefinition(validFilePath)
	if err != nil {
		t.Errorf("unexpected error parsing valid file: %v", err)
	}
	if taskDef.Family != "test-app" {
		t.Errorf("expected family 'test-app', got '%s'", taskDef.Family)
	}

	// Test non-existent file
	_, err = ParseTaskDefinition(filepath.Join(tmpDir, "nonexistent.json"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	// Test invalid JSON
	invalidFilePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFilePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write invalid test file: %v", err)
	}
	_, err = ParseTaskDefinition(invalidFilePath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestContainerDefinition_GetEffectiveMemory(t *testing.T) {
	tests := []struct {
		name              string
		memory            int
		memoryReservation int
		want              int
	}{
		{
			name:              "memory set",
			memory:            512,
			memoryReservation: 256,
			want:              512,
		},
		{
			name:              "only memory reservation set",
			memory:            0,
			memoryReservation: 256,
			want:              256,
		},
		{
			name:              "neither set",
			memory:            0,
			memoryReservation: 0,
			want:              0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := &ContainerDefinition{
				Memory:            tt.memory,
				MemoryReservation: tt.memoryReservation,
			}
			if got := cd.GetEffectiveMemory(); got != tt.want {
				t.Errorf("GetEffectiveMemory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortMapping_GetEffectiveHostPort(t *testing.T) {
	tests := []struct {
		name          string
		containerPort int
		hostPort      int
		want          int
	}{
		{
			name:          "host port set",
			containerPort: 80,
			hostPort:      8080,
			want:          8080,
		},
		{
			name:          "host port not set",
			containerPort: 80,
			hostPort:      0,
			want:          80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PortMapping{
				ContainerPort: tt.containerPort,
				HostPort:      tt.hostPort,
			}
			if got := pm.GetEffectiveHostPort(); got != tt.want {
				t.Errorf("GetEffectiveHostPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortMapping_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pm      PortMapping
		wantErr bool
	}{
		{
			name: "valid tcp port",
			pm: PortMapping{
				ContainerPort: 80,
				HostPort:      8080,
				Protocol:      "tcp",
			},
			wantErr: false,
		},
		{
			name: "valid udp port",
			pm: PortMapping{
				ContainerPort: 53,
				Protocol:      "udp",
			},
			wantErr: false,
		},
		{
			name: "default protocol",
			pm: PortMapping{
				ContainerPort: 80,
			},
			wantErr: false,
		},
		{
			name: "invalid protocol",
			pm: PortMapping{
				ContainerPort: 80,
				Protocol:      "sctp",
			},
			wantErr: true,
		},
		{
			name: "container port zero",
			pm: PortMapping{
				ContainerPort: 0,
			},
			wantErr: true,
		},
		{
			name: "container port too high",
			pm: PortMapping{
				ContainerPort: 70000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pm.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
