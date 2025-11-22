package main

import (
	"net/http"

	"fizzbuzz/internal/data"
)

// fizzbuzzHandlerNoStats is a version without statistics for performance comparison
func (app *application) fizzbuzzHandlerNoStats(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		app.methodNotAllowedResponse(w, r)
		return
	}

	// Parse JSON request body into FizzBuzzInput struct
	var input data.FizzBuzzInput
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate the input parameters
	v := validateFizzBuzzInput(&input)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.ErrorMap())
		return
	}

	// Execute the FizzBuzz algorithm using existing implementation
	result := data.FizzBuzz(input.Int1, input.Int2, input.Limit, input.Str1, input.Str2)

	// Create output struct with result
	output := data.FizzBuzzOutput{
		Result: result,
	}

	// Return success response using JSON envelope format
	err = app.writeJSON(w, http.StatusOK, envelope{"data": output}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
