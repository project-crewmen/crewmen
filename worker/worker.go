package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"

	"crewmen/stats"
	"crewmen/store"
	"crewmen/task"
)

// Track on the tasks

type Worker struct {
	Name      string
	Queue     queue.Queue  // Used to accept a task from manager (FIFO order follows)
	Db        store.Store  // In-memory DB: Used to track tasks
	Stats     *stats.Stats // Statistics
	TaskCount int          // Number of task operate by the worker at runtime
}

func New(name string, taskDbType string) *Worker {
	w := Worker{
		Name:  name,
		Queue: *queue.New(),
	}

	var s store.Store
	var err error

	switch taskDbType {
	case "memory":
		s = store.NewInMemoryTaskStore()
	case "persistent":
		fileName := fmt.Sprintf("%s_tasks.db", name)
		s, err = store.NewTaskStore(fileName, 0600, "tasks")
	}

	if err != nil {
		log.Printf("unable to create new task store: %v", err)
	}

	w.Db = s
	return &w
}

func (w *Worker) GetTasks() []*task.Task {
	taskList, err := w.Db.List()

	if err != nil {
		log.Printf("error getting list of tasks: %v", err)
		return nil
	}

	return taskList.([]*task.Task)
}

// Regularly collect metrics
func (w *Worker) CollectStats() {
	// This for loop indicates infinite loop with delay of 15 seconds
	for {
		log.Println("Collecting stats")
		w.Stats = stats.GetStats()
		w.TaskCount = w.Stats.TaskCount
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) RunTasks() {
	// Continuos loop that checks worker's queue for tasks
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("No tasks to process currently.\n")
		}

		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func (w *Worker) runTask() task.DockerResult {
	// Pull a task from the worker's queue
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("[worker] No tasks in the queue")

		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)
	fmt.Printf("[worker] Found task in queue: %v:\n", taskQueued)

	err := w.Db.Put(taskQueued.ID.String(), &taskQueued)
	if err != nil {
		msg := fmt.Errorf("error storing task %s: %v", taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	result, err := w.Db.Get(taskQueued.ID.String())
	if err != nil {
		msg := fmt.Errorf("error storing task %s from database: %v", taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	taskPersisted := *result.(*task.Task)

	if taskPersisted.State == task.Completed {
		return w.StopTask(taskPersisted)
	}

	var dockerResult task.DockerResult

	// Check whether the state transition is valid or not
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		// If the task from the queue in Scheduled state, then Start the task
		case task.Scheduled:
			if taskQueued.ContainerID != "" {
				dockerResult = w.StopTask(taskQueued)
				if dockerResult.Error != nil {
					log.Printf("%v\n", dockerResult.Error)
				}
			}

			dockerResult = w.StartTask(taskQueued)
		default:
			fmt.Printf("This is a mistake. taskPersisted: %v, taskQueued: %v\n", taskPersisted, taskQueued)
			dockerResult.Error = errors.New("we should not get here")
		}
	} else {
		// Return error if it is an invalid transition
		err := fmt.Errorf("invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		dockerResult.Error = err
		return dockerResult
	}

	return dockerResult
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	// Update the start time of the task
	// t.StartTime = time.Now().UTC()
	// Create an instance of Docker struct to talk with Docker daemon via Docker SDK
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	// Try to start the task
	result := d.Run()
	if result.Error != nil {
		log.Printf("Err running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db.Put(t.ID.String(), &t)

		return result
	}

	// Update task meta data with new running container ID
	t.ContainerID = result.ContainerId
	t.State = task.Running
	// Save updated task t at worker's DB
	w.Db.Put(t.ID.String(), &t)

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	// Create an instance of Docker struct to talk with Docker daemon via Docker SDK
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	// Try to stop the container
	stopResult := d.Stop(t.ContainerID)
	if stopResult.Error != nil {
		log.Printf("%v\n", stopResult.Error)
	}

	// Try to remove the container
	removeResult := d.Remove(t.ContainerID)
	if removeResult.Error != nil {
		log.Printf("%v\n", removeResult.Error)
	}

	// Update the finish time of the task
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	// Save updated task t at worker's DB
	w.Db.Put(t.ID.String(), &t)
	log.Printf("Stopped and removed container %v for task %v", t.ContainerID, t.ID)

	return removeResult
}

// Health Checks
func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.Println("Checking status of tasks")
		w.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

// To check whether container started and running properly without any errors
// 1. Check the task is in Running state
// 2. Call InspectTask() for a task's container
// 3. If throws an error, flag task's state as Failed
func (w *Worker) updateTasks() {
	tasks, err := w.Db.List()
	if err != nil {
		log.Printf("error getting list of tasks: %v", err)
		return
	}

	for _, t := range tasks.([]*task.Task) {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				fmt.Printf("ERROR: %v", resp.Error)
			}

			if resp.Container == nil {
				log.Printf("No container for running task %s", t.ID)
				t.State = task.Failed
				w.Db.Put(t.ID.String(), t)
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("Container for task %s in non-running state %s", t.ID, resp.Container.State.Status)
				t.State = task.Failed
				w.Db.Put(t.ID.String(), t)
			}

			t.HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
			w.Db.Put(t.ID.String(), t)
		}
	}
}
