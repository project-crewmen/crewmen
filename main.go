package main

import (
	"fmt"
	"os"
	"strconv"
	
	"github.com/joho/godotenv"

	"crewmen/manager"
	"crewmen/worker"
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

	// Worker 1
	w1 :=  worker.New("worker-1", "persistent")
	wapi1 := worker.Api{Address: whost, Port: wport, Worker: w1}

	// Worker 2
	w2 := worker.New("worker-2", "persistent")
	wapi2 := worker.Api{Address: whost, Port: wport + 1, Worker: w2}

	// Worker 3
	w3 := worker.New("worker-3", "persistent")
	wapi3 := worker.Api{Address: whost, Port: wport + 2, Worker: w3}

	go w1.RunTasks()
	go w1.UpdateTasks()
	go wapi1.Start()

	go w2.RunTasks()
	go w2.UpdateTasks()
	go wapi2.Start()

	go w3.RunTasks()
	go w3.UpdateTasks()
	go wapi3.Start()

	fmt.Printf("------ Starting Crewmen Manager(%s:%d) ------\n", mhost, mport)

	workers := []string{
		fmt.Sprintf("%s:%d", whost, wport),
		fmt.Sprintf("%s:%d", whost, wport+1),
		fmt.Sprintf("%s:%d", whost, wport+2),
	}
	m := manager.New(workers, "roundrobin", "persistent")
	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.Start()
}
