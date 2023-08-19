package manager

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/project-crewmen/crewmen/task"
)

// Track on the workers

type Manager struct {
	Pending       queue.Queue                 // Populate when the task is initially submitted (FIFO order follows)
	TaskDb        map[string][]task.Task      // In-memory DB: Used to track tasks
	EventDb       map[string][]task.TaskEvent // In-memory DB: Used to track task events
	Workers       []string                    // Used to track workers in the cluster
	WorkerTaskMap map[string][]uuid.UUID      // Workers and their associated tasks
	TaskWorkerMap map[uuid.UUID]string        // Tasks and the relevant workers
}

func (m *Manager) SelectWorker() {
	fmt.Println("I will select an appropriate worker")
}

func (m *Manager) UpdateTasks() {
	fmt.Println("I will update tasks")
}

func (m *Manager) SelectTasks() {
	fmt.Println("I will select tasks")
}

func (m *Manager) SendWork() {
	fmt.Println("I will send work to the worker")
}
