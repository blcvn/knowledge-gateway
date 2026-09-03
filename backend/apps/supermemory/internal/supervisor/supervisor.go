package supervisor

import (
	"context"
	"log"
	"sort"
	"sync"
)

// Task represents a service or goroutine to be managed
type Task struct {
	Name     string
	Phase    int
	StartFn  func(context.Context) error
}

// Supervisor manages the lifecycle of tasks
type Supervisor struct {
	tasks []Task
	mu    sync.Mutex
}

func New() *Supervisor {
	return &Supervisor{
		tasks: make([]Task, 0),
	}
}

// Register adds a task to the supervisor
func (s *Supervisor) Register(t Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = append(s.tasks, t)
}

// Start executes all tasks in order of their phase
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Sort tasks by phase ascending
	sort.Slice(s.tasks, func(i, j int) bool {
		return s.tasks[i].Phase < s.tasks[j].Phase
	})

	for _, task := range s.tasks {
		log.Printf("Starting task %s (Phase %d)", task.Name, task.Phase)
		go func(t Task) {
			if err := t.StartFn(ctx); err != nil {
				log.Printf("Task %s failed: %v", t.Name, err)
			}
		}(task)
	}

	return nil
}
