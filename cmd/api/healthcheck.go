package main

import (
	"net/http"
)

func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	responseData := map[string]string{
		"status":  "available",
		"state":   app.config.state,
		"version": version,
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"system_info": responseData}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
