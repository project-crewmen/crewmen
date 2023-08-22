package task

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

/*
Define the state of a task's life cycle.
-- Initial State is Pending.
-- When its scheduled to be run, then its in the Scheduled state.
-- If the machine successfully started the task, then its in the Running state.
-- Once task executed successfully, its in Completed state.
-- If task crashes at some point then its Failed state.
*/
type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	// Main attributes of a task
	ID    uuid.UUID
	Name  string
	State State
	// Associated docker container related metadata
	Image         string
	Memory        int
	Disk          int
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
}

// To handle the events related to Tasks. (Usually to swtich the task states)
type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

// Holds configurations for orchestration tasks
type Config struct {
	Name          string // To Idenity a task in the Orch Sys
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	Cmd           []string
	Image         string   // Image name that container runs
	Memory        int64    // To tell docker-daemon about the memory required for a task
	Disk          int64    // To tell docker-daemon about the space required for a task
	Env           []string // Specify environment variables that passed into the container
	RestartPolicy string   // Specify to docker-daemon what to do when the container fails
}

// Specifications to run a task as a Docker container
type Docker struct {
	Client      *client.Client
	Config      Config
	ContainerId string
}

// Docker info wrapper for common results
type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}

// Pull Image from Dockerhub and run the container as a task
func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, types.ImagePullOptions{})

	if err != nil {
		log.Printf("Error pulling image %s: %v\n", d.Config.Image, err)

		return DockerResult{Error: err}
	}

	io.Copy(os.Stdout, reader)

	//  Config info that passes to ContainerCreate
	rp := container.RestartPolicy{
		Name: d.Config.RestartPolicy,
	}

	r := container.Resources{
		Memory: d.Config.Memory,
	}

	cc := container.Config{
		Image: d.Config.Image,
		Env:   d.Config.Env,
	}

	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)

	if err != nil {
		log.Printf("Error creating container using image %s: %v\n", d.Config.Image, err)

		return DockerResult{Error: err}
	}

	// CHANGED := to =
	err = d.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})

	if err != nil {
		log.Printf("Error starting container using image %s: %v\n", resp.ID, err)

		return DockerResult{Error: err}
	}

	// Logging
	// CHANGED d.Config.Runtime.ContainerId to d.ContainerId
	d.ContainerId = resp.ID

	// Changed cli to Container logs
	out, err := d.Client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})

	if err != nil {
		log.Printf("Error getting logs for container %s: %v\n", resp.ID, err)

		return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return DockerResult{
		ContainerId: resp.ID,
		Action:      "Start",
		Result:      "success",
	}
}

// Stop Container
// CHANGED: removed id string parameter
func (d *Docker) Stop() DockerResult {
	log.Printf("Attempting to stop container %v", d.ContainerId)

	ctx := context.Background()
	// Changed nil to container.StopOptions{}
	noWaitTimeout := 0 // to not wait for the container to exit gracefully
	err := d.Client.ContainerStop(ctx, d.ContainerId, container.StopOptions{ Timeout: &noWaitTimeout})

	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	err = d.Client.ContainerRemove(ctx, d.ContainerId, types.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: false})

	if err != nil {
		panic(err)
	}

	return DockerResult{Action: "stop", Result: "success", Error: nil}
}
