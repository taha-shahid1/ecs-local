package docker

// Client wraps the Docker SDK client
type Client struct {
	// Docker client will be added here
}

// NewClient creates a new Docker client
func NewClient() (*Client, error) {
	return &Client{}, nil
}
