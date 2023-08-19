package worker

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/project-crewmen/crewmen/task"
)

// Track on the tasks

type Worker struct {
	Name      string
	Queue     queue.Queue             // Used to accept a task from manager (FIFO order follows)
	Db        map[uuid.UUID]task.Task // In-memory DB: Used to track tasks
	TaskCount int                     // Number of task operate by the worker at runtime
}

func (w *Worker) CollectStats() {
	fmt.Println("I will collect stats")
}

func (w *Worker) RunTask() {
	fmt.Println("I will start or stop a task")
}

func (w *Worker) StartTask() {
	fmt.Println("I will start a task")
}

func (w *Worker) StopTask() {
	fmt.Println("I will stop a task")
}
