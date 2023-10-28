# Crewmen - Multi-Constrained Container Orchestrator
<img src="./brand/Logo.png" alt="Crewmen Logo" style="width: 200px;">

Official Web Site - <a href="https://crewmen.vercel.app/" target="_blank">Crewmen</a>


## Installation
1. Install Go lang in your device
2. Open the command line in the project directory and execute `go run main.go`

## Components
### Task
* Specify metadata related to memory, CPU, disk etc.
* Restart policy - Orchestrator will take an action when a task fails according to the restart policy
* Specify metadata telated to container such as image.

### Job

### Scheduler
* Determine a set of candidate workers that can run a task
* Score the candidate workers from best to worst
* Pick the worker with the best score

### Manager
* Accept user request to start/stop tasks
* Schedule tasks onto worker machines
* Monitor tasks, their states and worker machines which they run

### Worker
* Run tasks as Docker containers.
* Accepting tasks to run from manager.
* Provide statistics to manager for task scheduling.
* Monitor the associated tasks and their states.

### Cluster

### CLI

## Debug Docker
* **Docker Image Pull Permissions** - Got permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock: Get http://%2Fvar%2Frun%2Fdocker.sock/v1.40/containers/json: dial unix /var/run/docker.sock: connect: permission denied
`sudo chmod 666 /var/run/docker.sock`

## Go Helpers
How to download and import a new go module?
`go get MODULE_NAME`