package config

// Config holds the configuration for ecs-dev
type Config struct {
	DockerHost string
	LogLevel   string
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		DockerHost: "unix:///var/run/docker.sock",
		LogLevel:   "info",
	}
}
