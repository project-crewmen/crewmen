package main

import (
	"fmt"
	"log"

	// "os"
	"time"
	// "strconv"

	// "github.com/docker/docker/client"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	// "github.com/project-crewmen/crewmen/manager"
	// "github.com/project-crewmen/crewmen/node"
	"github.com/project-crewmen/crewmen/manager"
	"github.com/project-crewmen/crewmen/task"
	"github.com/project-crewmen/crewmen/worker"
)

func main() {
	host := "localhost"
	port := 8080

	fmt.Printf("Starting the crewmen on  %v:%v\n", host, port)

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.Api{Address: host, Port: port, Worker: &w}

	// Runs on a seperate go routines
	go runTasks(&w)
	// go w.CollectStats()
	go api.Start()

	// Instantiating a manager
	workers := []string{fmt.Sprintf("%s:%d", host, port)}
	m := manager.New(workers)

	// Add three tasks to the manager
	for i := 0; i < 3; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}

		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}

		m.AddTask(te)
		m.SendWork()
	}

	go func() {
		for {
			fmt.Printf("[Manager] Updating tasks from %d workers\n", len(m.Workers))
			m.UpdateTasks()
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		for _, t := range m.TaskDb {
			fmt.Printf("[Manager] Task--> id: %s, state: %d\n", t.ID, t.State)
			time.Sleep(15 * time.Second)
		}
	}
}

func runTasks(w *worker.Worker) {
	// Continuos loop that checks worker's queue for tasks
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
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
