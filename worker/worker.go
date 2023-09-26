package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/project-crewmen/crewmen/task"
)

// Track on the tasks

type Worker struct {
	Name      string
	Queue     queue.Queue              // Used to accept a task from manager (FIFO order follows)
	Db        map[uuid.UUID]*task.Task // In-memory DB: Used to track tasks
	Stats     *Stats                   // Statistics
	TaskCount int                      // Number of task operate by the worker at runtime
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := []*task.Task{}

	for _, t := range w.Db {
		tasks = append(tasks, t)
	}

	return tasks
}

// Regularly collect metrics
func (w *Worker) CollectStats() {
	// This for loop indicates infinite loop with delay of 15 seconds
	for {
		log.Println("Collecting stats")
		w.Stats = GetStats()
		w.TaskCount = w.Stats.TaskCount
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) RunTask() task.DockerResult {
	// Pull a task from the worker's queue
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("No tasks in the queue")

		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)

	// Get the task from worker's DB
	taskPersisted := w.Db[taskQueued.ID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskPersisted.ID] = &taskQueued
	}

	var result task.DockerResult

	// Check whether the state transition is valid or not
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		// If the task from the queue in Scheduled state, then Start the task
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		// If the task from the queue in Completed state, then Stop the task
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("we should not get here")
		}
	} else {
		// Return error if it is an invalid transition
		err := fmt.Errorf("invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}

	return result
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	// Update the start time of the task
	t.StartTime = time.Now().UTC()
	// Create an instance of Docker struct to talk with Docker daemon via Docker SDK
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	// Try to start the task
	result := d.Run()
	if result.Error != nil {
		log.Printf("Err running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t

		return result
	}

	// Update task meta data with new running container ID
	t.ContainerID = result.ContainerId
	t.State = task.Running
	// Save updated task t at worker's DB
	w.Db[t.ID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	// Create an instance of Docker struct to talk with Docker daemon via Docker SDK
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	// Try to stop the task
	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf("Error stopping container %v: %v", t.ContainerID, result.Error)
	}

	// Update the finish time of the task
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	// Save updated task t at worker's DB
	w.Db[t.ID] = &t
	log.Printf("Stopped and removed container %v for task %v", t.ContainerID, t.ID)

	return result
}
