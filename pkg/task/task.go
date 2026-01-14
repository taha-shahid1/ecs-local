package task

// Task represents a running ECS task
type Task struct {
	ID          string
	Name        string
	Containers  []string
	Status      string
	StartedAt   string
}

// Manager handles task lifecycle
type Manager struct {
	// Will hold docker client and task state
}

// NewManager creates a new task manager
func NewManager() *Manager {
	return &Manager{}
}
