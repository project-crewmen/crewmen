package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/project-crewmen/crewmen/task"
	"github.com/project-crewmen/crewmen/worker"
)

// Track on the workers

type Manager struct {
	Pending       queue.Queue                   // Populate when the task is initially submitted (FIFO order follows)
	TaskDb        map[uuid.UUID]*task.Task      // In-memory DB: Used to track tasks
	EventDb       map[uuid.UUID]*task.TaskEvent // In-memory DB: Used to track task events
	Workers       []string                      // Used to track workers in the cluster
	WorkerTaskMap map[string][]uuid.UUID        // Workers and their associated tasks
	TaskWorkerMap map[uuid.UUID]string          // Tasks and the relevant workers
	LastWorker    int
}

func New(workers []string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)
	workerTaskmap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	for worker := range workers {
		workerTaskmap[workers[worker]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskmap,
		TaskWorkerMap: taskWorkerMap,
	}
}

func (m *Manager) SelectWorker() string {
	var newWorker int

	// Select and assign a worker in round robin fashion
	if m.LastWorker+1 < len(m.Workers) {
		newWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		newWorker = 0
		m.LastWorker = 0
	}

	return m.Workers[newWorker]
}

/* Periodically get the status update from workers to update the manager's state */
func (m *Manager) UpdateTasks() {
	for _, worker := range m.Workers {
		// Query each worker and get their own tasks list
		log.Printf("Checking worker %v for task updates\n", worker)

		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)

		if err != nil {
			log.Printf("Error connectinh to %v: %v\n", worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error sending request: %v\n", err)
		}

		// Decode the response received from a worker
		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)

		if err != nil {
			log.Printf("Error unmarshalling tasks: %s\n", err.Error())
		}

		// Go through all the tasks of a worker
		for _, t := range tasks {
			log.Printf("Attempting to update task %v\n", t.ID)

			_, ok := m.TaskDb[t.ID]

			if !ok {
				log.Printf("Task with ID %s not found\n", t.ID)
				return
			}

			// Check if the task's state in the manager and worker is same or not
			if m.TaskDb[t.ID].State != t.State {
				m.TaskDb[t.ID].State = t.State
			}

			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) SendWork() {
	// Check if there are task events in the pending queue
	if m.Pending.Len() > 0 {
		// Select a worker to run a task
		w := m.SelectWorker()

		// Pull a task from pending queue
		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)
		t := te.Task
		log.Printf("Pulled %v off pending queue\n", t)

		// Update managers track
		m.EventDb[te.ID] = &te
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)
		m.TaskWorkerMap[t.ID] = w

		// Set the state of the task to be scheduled
		t.State = task.Scheduled
		m.TaskDb[t.ID] = &t

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf("Unable to marshal task object: %v\n", t)
		}

		// JSON encode the task event and send it to the selected worker
		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

		if err != nil {
			log.Printf("Error connecting to %v: %v\n", w, err)
			m.Pending.Enqueue(te)
			return
		}

		// Decord and check the response from the worker
		d := json.NewDecoder(resp.Body)

		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrorResponse{}
			err := d.Decode(&e)

			if err != nil {
				fmt.Printf("Error decoding response: %s\n", err.Error())
				return
			}

			log.Printf("Response error (%d): %s\n", e.HTTPStatusCode, e.Message)
			return
		}

		t = task.Task{}
		err = d.Decode(&t)

		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}

		log.Printf("%#v\n", t)
	} else {
		log.Printf("No work in the queue")
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	// Enqueue a task to the manager's pending queue
	m.Pending.Enqueue(te)
}
