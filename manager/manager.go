package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"crewmen/node"
	"crewmen/scheduler"
	"crewmen/task"
	"crewmen/worker"
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
	WorkerNodes   []*node.Node        // Instance of eachc worker nodes
	Scheduler     scheduler.Scheduler // Manager can use any scheduler that implements this interface
}

func New(workers []string, schedulerType string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)

	workerTaskmap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	// Creating nodes
	var nodes []*node.Node
	for worker := range workers {
		workerTaskmap[workers[worker]] = []uuid.UUID{}

		// Refer a worker's API. For instance http://192.168.1.1:5555
		nAPI := fmt.Sprintf("http://%v", workers[worker])
		n := node.NewNode(workers[worker], nAPI, "worker")
		nodes = append(nodes, n)
	}

	// Selecting the scheduler
	var s scheduler.Scheduler
	switch schedulerType {
	case "roundrobin":
		s = &scheduler.RoundRobin{Name: "roundrobin"}
	case "greedy":
		s = &scheduler.Greedy{Name: "greedy"}
	default:
		s = &scheduler.Epvm{Name: "epvm"}
	}

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskmap,
		TaskWorkerMap: taskWorkerMap,
		WorkerNodes:   nodes,
		Scheduler:     s,
	}
}

func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	// 1. Select candidate nodes for the task
	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)
	if candidates == nil {
		msg := fmt.Sprintf("No available candidates match resource request for task %v", t.ID)
		err := errors.New(msg)

		return nil, err
	}

	// 2. Score candidates
	scores := m.Scheduler.Score(t, candidates)
	if scores == nil {
		return nil, fmt.Errorf("no scores returned to task %v", t)
	}

	// 3. Pick a node to run the task
	selectedNode := m.Scheduler.Pick(scores, candidates)

	return selectedNode, nil
}

func (m *Manager) UpdateTasks() {
	for {
		log.Println("Checking for task updates from workers")
		m.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

/* Periodically get the status update from workers to update the manager's state */
func (m *Manager) updateTasks() {
	for _, worker := range m.Workers {
		// Query each worker and get their own tasks list
		log.Printf("Checking worker %v for task updates\n", worker)

		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)

		if err != nil {
			log.Printf("Error connecting to %v: %v\n", worker, err)
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

// Endless loop to Send work for workers via manager
func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing any tasks in the queue")
		m.SendWork()
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) stopTask(worker string, taskID string){
	client := &http.Client{}
	url := fmt.Sprintf("http://%s/tasks/%s", worker, taskID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("error creating request to delete task %s: %v", taskID, err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error connecting to worker at %s: %v", url, err)
		return
	}

	if resp.StatusCode != 204 {
		log.Printf("Error sending request: %v", err)
		return
	}

	log.Printf("task %s has been scheduled to be stopped", taskID)
}

func (m *Manager) SendWork() {
	// Check if there are task events in the pending queue
	if m.Pending.Len() > 0 {
		// Pull a task from pending queue
		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)
		m.EventDb[te.ID] = &te
		log.Printf("Pulled %v off pending queue\n", te)

		taskWorker, ok := m.TaskWorkerMap[te.Task.ID]
		if ok {
			persistedTask := m.TaskDb[te.Task.ID]
			if te.State == task.Completed && task.ValidStateTransition(persistedTask.State, te.State) {
				m.stopTask(taskWorker, te.Task.ID.String())
				return
			}

			log.Printf("invalid request: existing task %s is in state %v and cannot transition to the completed state", persistedTask.ID.String(), persistedTask.State)
			return
		}

		t := te.Task
		w, err := m.SelectWorker(t)
		if err != nil {
			log.Printf("error selecting worker for task %s: %v", t.ID, err)
			return
		}

		log.Printf("[manager] selected worker %s for task %s", w.Name, t.ID)

		m.WorkerTaskMap[w.Name] = append(m.WorkerTaskMap[w.Name], te.Task.ID)
		m.TaskWorkerMap[t.ID] =w.Name

		t.State = task.Scheduled
		m.TaskDb[t.ID] = &t

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf("Unable to marshal task object: %v\n", t)
		}

		// JSON encode the task event and send it to the selected worker
		url := fmt.Sprintf("http://%s/tasks", w.Name)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

		if err != nil {
			log.Printf("[manager] Error connecting to %v: %v\n", w, err)
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

		w.TaskCount++
		log.Printf("[manager] received response from worker: %#v\n", t)
	} else {
		log.Printf("No work in the queue")
	}
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}

	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}

	return tasks
}

func (m *Manager) AddTask(te task.TaskEvent) {
	// Enqueue a task to the manager's pending queue
	m.Pending.Enqueue(te)
}

// Health Checks
func (m *Manager) DoHealthChecks() {
	for {
		log.Println("Performing task health check")
		m.doHealthChecks()
		log.Println("Task health checks completed")
		log.Println("Sleeping for 60 seconds")
		time.Sleep(60 * time.Second)
	}
}

// Perform health check for each tasks
// RestartCount = 3 (TODO: Make this more robust when crewmen is production-ready)
func (m *Manager) doHealthChecks() {
	for _, t := range m.TaskDb {
		// 1. Task should be running and max attempts to capture health is 3
		// ---> If not errors then health is good
		// ---> if has error then restart the task
		// 2. If task is failed and not exceeded the maximum restartCount, attempt to restart the task
		if t.State == task.Running && t.RestartCount < 3 {
			err := m.checkTaskHealth(*t)
			if err != nil {
				if t.RestartCount < 3 {
					m.restartTask(t)
				}
			}
		} else if t.State == task.Failed && t.RestartCount < 3 {
			m.restartTask(t)
		}
	}
}

func (m *Manager) restartTask(t *task.Task) {
	// Look for the worker in the TaskWorkerMap of Manager
	w := m.TaskWorkerMap[t.ID]
	// Flag the task as scheduled, because its going to be restarted
	t.State = task.Scheduled
	t.RestartCount++

	// Overwrite the existing task in the TaskDb
	m.TaskDb[t.ID] = t

	// Create a TaskEvent, Marshal as a JSON payload and send it to the relevant worker
	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}

	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("Uanble to marshal task object: %v", t)
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error connecting to %v: %v", w, err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		e := worker.ErrorResponse{}
		err := d.Decode(&e)
		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}

		log.Printf("Response error (%d): %s", e.HTTPStatusCode, e.Message)
		return
	}

	newTask := task.Task{}
	err = d.Decode(&newTask)
	if err != nil {
		fmt.Printf("Error decoding response: %s\n", err.Error())
		return
	}

	log.Printf("%#v\n", t)
}

func getHostPort(ports nat.PortMap) *string {
	for k, _ := range ports {
		return &ports[k][0].HostPort
	}

	return nil
}

func (m *Manager) checkTaskHealth(t task.Task) error {
	log.Printf("Calling health check for task %s: %s\n", t.ID, t.HealthCheck)

	// Each worker address in the TaskWorkerMap follows the format IP:PORT. For instance (192.168.1.1:1234)
	w := m.TaskWorkerMap[t.ID]
	hostPort := getHostPort(t.HostPorts)
	worker := strings.Split(w, ":")

	if hostPort == nil {
		log.Printf("Have not collected task %s host port yet. Skipping.\n", t.ID)
		return nil
	}

	// Build a URL to monitor worker's health follows the format http://HOST_IP:HOST_PORT/HealthCheck (For instance http://192.168.1.1:1234/health)
	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostPort, t.HealthCheck)
	log.Printf("Calling health check for task %s: %s\n", t.ID, url)
	resp, err := http.Get(url)

	if err != nil {
		msg := fmt.Sprintf("Error connecting to health check %s", url)
		log.Println(msg)

		return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Error health check for task %s did not return 200\n", t.ID)
		log.Println(msg)

		return errors.New(msg)
	}

	log.Printf("Task %s health check response: %v\n", t.ID, resp.StatusCode)

	// If return nil Then no error in the container, Which means health is good
	return nil
}
