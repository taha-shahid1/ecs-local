package config

// Config represents ecs-dev configuration
type Config struct {
	DockerHost string
	LogLevel   string
}

// NewConfig returns default configuration
func NewConfig() *Config {
	return &Config{
		DockerHost: "unix:///var/run/docker.sock",
		LogLevel:   "info",
	}
}
