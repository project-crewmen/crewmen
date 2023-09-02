package main

import (
	"fmt"
	// "os"
	"time"

	// "github.com/docker/docker/client"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	// "github.com/project-crewmen/crewmen/manager"
	// "github.com/project-crewmen/crewmen/node"
	"github.com/project-crewmen/crewmen/task"
	"github.com/project-crewmen/crewmen/worker"
)

func main() {
	db := make(map[uuid.UUID]*task.Task)
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    db,
	}

	t := task.Task{
		ID:    uuid.New(),
		Name:  "test-container-1",
		State: task.Scheduled,
		Image: "strm/helloworld-http",
	}

	fmt.Println("starting the task")
	w.AddTask(t)
	result := w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}

	t.ContainerID = result.ContainerId

	fmt.Printf("task %s is running in container %s\n", t.ID, t.ContainerID)
	fmt.Println("sleep time")
	time.Sleep(time.Second * 60)

	fmt.Printf("stopping the task %s\n", t.ID)
	t.State = task.Completed
	w.AddTask(t)
	result = w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}
}

// func main() {
// 	// Create a task
// 	t := task.Task{
// 		ID:     uuid.New(),
// 		Name:   "Task-1",
// 		State:  task.Pending,
// 		Image:  "Image-1",
// 		Memory: 1024,
// 		Disk:   1,
// 	}

// 	// Create a task event
// 	te := task.TaskEvent{
// 		ID:        uuid.New(),
// 		State:     task.Pending,
// 		Timestamp: time.Now(),
// 		Task:      t,
// 	}

// 	// Print task and taskevent objects
// 	fmt.Printf("task: %v\n", t)
// 	fmt.Printf("task event: %v\n", te)

// 	// Create a worker
// 	w := worker.Worker{
// 		Queue: *queue.New(),
// 		Db:    make(map[uuid.UUID]*task.Task),
// 	}

// 	// Print the worker object
// 	fmt.Printf("worker: %v\n", w)

// 	// Call the worker's methods
// 	w.CollectStats()
// 	w.RunTask()
// 	w.StartTask(t)
// 	w.StopTask(t)

// 	// Create a manager
// 	m := manager.Manager{
// 		Pending: *queue.New(),
// 		TaskDb:  make(map[string][]task.Task),
// 		EventDb: make(map[string][]task.TaskEvent),
// 		Workers: []string{w.Name},
// 	}

// 	// Print the manager object
// 	fmt.Printf("manager: %v\n", m)

// 	// Call the manager's methods
// 	m.SelectWorker()
// 	m.UpdateTasks()
// 	m.SelectTasks()
// 	m.SendWork()

// 	// Create a node
// 	n := node.Node{
// 		Name:   "Node-1",
// 		Ip:     "192.168.1.1",
// 		Cores:  4,
// 		Memory: 1024,
// 		Disk:   25,
// 		Role:   "worker",
// 	}

// 	// Print the node object
// 	fmt.Printf("node: %v\n", n)

// 	// DOCKER CONTAINER RELATED
// 	fmt.Printf("create a test container\n")
// 	dockerTask, createResult := createContainer()

// 	if createResult.Error != nil {
// 		fmt.Printf(createResult.Error.Error())
// 		os.Exit(1)
// 	}

// 	time.Sleep(time.Second * 5)
// 	fmt.Printf("stopping container %s\n", createResult.ContainerId)
// 	_ = stopContainer(dockerTask)
// }

// func createContainer() (*task.Docker, *task.DockerResult) {
// 	c := task.Config{
// 		Name:  "test-container-1",
// 		Image: "postgres:13",
// 		Env: []string{
// 			"POSTGRES_USER=cube",
// 			"POSTGRES_PASSWORD=secret",
// 		},
// 	}

// 	dc, _ := client.NewClientWithOpts(client.FromEnv)

// 	d := task.Docker{
// 		Client: dc,
// 		Config: c,
// 	}

// 	result := d.Run()

// 	if result.Error != nil {
// 		fmt.Printf("%v\n", result.Error)
// 	}

// 	fmt.Printf("Container %s is running with config %v\n", result.ContainerId, c)

// 	return &d, &result
// }

// func stopContainer(d *task.Docker) *task.DockerResult {
// 	fmt.Printf("containerID: %v\n", d)

// 	result := d.Stop()

// 	if result.Error != nil {
// 		fmt.Printf("%v\n", result.Error)

// 		return nil
// 	}

// 	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerId)

// 	return &result
// }
