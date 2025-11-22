package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Check Content-Type header for POST requests to API endpoints
	if r.Method == http.MethodPost && r.URL != nil && strings.HasPrefix(r.URL.Path, "/v1/") {
		ct := r.Header.Get("Content-Type")
		if ct == "" {
			return errors.New("missing Content-Type header")
		}
		if ct != "application/json" {
			return errors.New("Content-Type header is not application/json")
		}
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return errors.New("the request body contains badly-formed JSON")

		case err.Error() == "unexpected EOF":
			return errors.New("the request body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return errors.New("the request body contains an invalid value for the \"" + unmarshalTypeError.Field + "\" field")
			}
			return errors.New("the request body contains invalid JSON data types")

		case strings.Contains(err.Error(), "missing brace"):
			return errors.New("the request body contains badly-formed JSON")

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		case errors.As(err, &maxBytesError):
			return errors.New("the request body is too large")

		case strings.Contains(err.Error(), "request body too large"):
			return errors.New("the request body is too large")

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != nil && err.Error() != "EOF" {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (app *application) errorJSON(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}

	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logger.Error("error writing JSON response",
			"error", err,
			"status", status,
			"response_type", "error_json")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error("server error",
		"error", err,
		"method", r.Method,
		"uri", r.URL.RequestURI(),
		"addr", r.RemoteAddr)
	message := "the server encountered a problem and could not process your request"
	app.errorJSON(w, r, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorJSON(w, r, http.StatusNotFound, message)
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	// Set Allow header based on the requested path
	switch r.URL.Path {
	case "/v1/fizzbuzz":
		w.Header().Set("Allow", "POST")
	case "/v1/healthcheck", "/v1/statistics":
		w.Header().Set("Allow", "GET")
	default:
		w.Header().Set("Allow", "GET, POST")
	}

	message := "the " + r.Method + " method is not supported for this resource"
	app.errorJSON(w, r, http.StatusMethodNotAllowed, message)
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorJSON(w, r, http.StatusBadRequest, err.Error())
}

func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorJSON(w, r, http.StatusUnprocessableEntity, map[string]any{
		"message": "validation failed",
		"details": errors,
	})
}

func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	app.errorJSON(w, r, status, message)
}
