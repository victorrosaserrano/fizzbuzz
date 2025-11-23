package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/fizzbuzz", app.fizzbuzzHandler)
	router.HandlerFunc(http.MethodGet, "/v1/statistics", app.statisticsHandler)

	return app.correlationID(app.logRequest(app.rateLimit(app.rateLimiter)(app.recoverPanic(router))))
}

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	app.logger.InfoWithContext(r.Context(), "health check requested")

	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     "1.0.0",
		},
	}

	err := app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
