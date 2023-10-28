package worker

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ErrorResponse struct {
	HTTPStatusCode int
	Message        string
}

// Used to define the Address and Port that defines the local IP address of the Machine
type Api struct {
	Address string
	Port    int
	// Pointer to other types in the crewmen
	Worker *Worker  // Reference to a Worker object to exprose its services via API
	Router *chi.Mux // Expose the routing features of chi package
}

func (a *Api) initRouter() {
	a.Router = chi.NewRouter()

	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)

		r.Route("/{taskID}", func(r chi.Router){
			r.Delete("/", a.StopTaskHandler)
			r.Get("/", a.InspectTaskHandler)	// Health Check Inspection
		})
	})

	a.Router.Route("/stats", func(r chi.Router) {
		r.Get("/", a.GetStatsHandler)
	})
}

func (a *Api) Start() {
	a.initRouter()
	fmt.Println("Worker API Started")
	http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
}
