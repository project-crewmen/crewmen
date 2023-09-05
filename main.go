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

	// Runs on a seperate go routine
	go runTasks(&w)
	api.Start()
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
