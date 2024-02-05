package router

import (
	"net/http"

	handler "example.com/Go/internal/transport/handler"
)

func RegisterRoutes() {
	http.HandleFunc("/people", handler.PeopleHandler)
	http.HandleFunc("/health", handler.HealthCheckHandler)
}
