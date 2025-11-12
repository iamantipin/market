package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthCheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/ads/:id", app.requirePermission("ads:read", app.showAdHandler))
	router.HandlerFunc(http.MethodGet, "/v1/ads", app.requirePermission("ads:read", app.showAdsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/ads", app.requirePermission("ads:write", app.postAdHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/ads/:id", app.requirePermission("ads:write", app.updateAdHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/ads/:id", app.requirePermission("ads:write", app.deleteAdHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.rateLimit(app.authenticate(router)))
}
