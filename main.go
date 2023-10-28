package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"crewmen/manager"
	"crewmen/task"
	"crewmen/worker"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	envErr := godotenv.Load(".env")
	if envErr != nil {
		fmt.Printf("Could not load .env file")
		os.Exit(1)
	}

	whost := os.Getenv("CREWMEN_WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("CREWMEN_WORKER_PORT"))

	mhost := os.Getenv("CREWMEN_MANAGER_HOST")
	mport, _ := strconv.Atoi(os.Getenv("CREWMEN_MANAGER_PORT"))

	fmt.Printf("------ Starting Crewmen Worker(%s:%d) ------\n", whost, wport)

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	wapi := worker.Api{Address: whost, Port: wport, Worker: &w}

	go w.RunTasks()
	// go w.CollectStats()
	go w.UpdateTasks()
	go wapi.Start()

	fmt.Printf("------ Starting Crewmen Manager(%s:%d) ------\n", mhost, mport)

	workers := []string{fmt.Sprintf("%s:%d", whost, wport)}
	m := manager.New(workers)
	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.Start()
}
