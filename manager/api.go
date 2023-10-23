package manager

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
	Manager *Manager  // Reference to a Worker object to exprose its services via API
	Router *chi.Mux // Expose the routing features of chi package
}

func (a *Api) initRouter() {
	a.Router = chi.NewRouter()

	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Delete("/{taskID}", a.StopTaskHandler)
	})
}

func (a *Api) Start() {
	a.initRouter()
	fmt.Println("Manager API Started")
	http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
}