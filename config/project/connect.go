package project

import (
	"fmt"
	"sync"

	"knowledge-srv/config"
	"knowledge-srv/pkg/project"
)

var (
	instance *project.Client
	once     sync.Once
	mu       sync.RWMutex
)

// Connect initializes the Project Service client using singleton pattern.
func Connect(cfg config.ProjectConfig) *project.Client {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance
	}

	once.Do(func() {
		// pkg/project/client.go NewClient(baseURL string)
		instance = project.NewClient(cfg.URL)
	})

	return instance
}

// GetClient returns the singleton Project Service client instance.
func GetClient() *project.Client {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("Project client not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if Project Service client is initialized
func HealthCheck() error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("Project client not initialized")
	}
	return nil
}
