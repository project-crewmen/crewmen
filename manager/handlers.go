package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"crewmen/task"
)

// For Tasks
func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Read the body of the request
	d := json.NewDecoder(r.Body)
	// d.DisallowUnknownFields()

	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Printf(msg)
		w.WriteHeader(400) // HTTP Status Code 400 - Bad Request
		e := ErrorResponse{
			HTTPStatusCode: 400,
			Message:        msg,
		}
		json.NewEncoder(w).Encode(e)

		return
	}

	a.Manager.AddTask(te)
	log.Printf("Added task %v\n", te.Task.ID)
	w.WriteHeader(201) // HTTP Status Code 201 - Successful POST request and resource created
	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200) // HTTP Status Code 200 - OK
	json.NewEncoder(w).Encode(a.Manager.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		log.Printf("No taskID \n")
		w.WriteHeader(400)
	}

	tID, _ := uuid.Parse(taskID)
	_, ok := a.Manager.TaskDb[tID]
	if !ok {
		log.Printf("No tasks with the taskID: %v found!", tID)
		w.WriteHeader(404)
	}

	te := task.TaskEvent{
		ID: uuid.New(),
		State: task.Completed,
		Timestamp: time.Now(),
	}

	taskToStop := a.Manager.TaskDb[tID]
	// Make a copy of task data, such that we are not modifying the actural task data on DB
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	te.Task = taskCopy
	a.Manager.AddTask(te)

	log.Printf("Added task %v to stop container %v\n", te.ID, taskToStop.ContainerID)
}