package task

import (
	"time"

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
