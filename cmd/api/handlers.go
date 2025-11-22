package main

import (
	"context"
	"net/http"
	"time"

	"fizzbuzz/internal/data"
	"fizzbuzz/internal/validator"
)

// fizzbuzzHandler handles POST requests to the /v1/fizzbuzz endpoint.
// It processes FizzBuzz requests by parsing the JSON input, executing the algorithm,
// and returning the result in the standard JSON envelope format.
func (app *application) fizzbuzzHandler(w http.ResponseWriter, r *http.Request) {
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

	// Story 4.6: Record statistics with context-aware PostgreSQL operations
	// Use defensive programming to ensure statistics failure doesn't affect response
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Log statistics recording failure but continue with response
				app.logger.Error("statistics recording failed",
					"error", r,
					"method", "POST",
					"uri", "/v1/fizzbuzz")
			}
		}()

		// Create context with timeout for statistics recording
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		err := app.statistics.Record(ctx, &input)
		if err != nil {
			// Log error but don't affect the main response
			app.logger.Warn("statistics recording failed",
				"error", err,
				"method", "POST",
				"uri", "/v1/fizzbuzz",
				"parameters", input)
		}
	}()

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

// statisticsHandler handles GET requests to the /v1/statistics endpoint.
// Returns the most frequently requested FizzBuzz parameters with hit count in JSON envelope format.
func (app *application) statisticsHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		app.methodNotAllowedResponse(w, r)
		return
	}

	// Story 4.6: Get most frequent statistics with context-aware PostgreSQL operations
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	mostFrequent, err := app.statistics.GetMostFrequent(ctx)
	if err != nil {
		app.logger.Error("failed to retrieve statistics",
			"error", err,
			"method", "GET",
			"uri", "/v1/statistics")
		app.serverErrorResponse(w, r, err)
		return
	}

	// Prepare response data structure
	var responseData envelope
	if mostFrequent == nil {
		// Empty statistics case: no requests processed yet
		responseData = envelope{
			"most_frequent_request": nil,
			"hits":                  0,
		}
	} else {
		// Populated statistics case: return most frequent parameters and hits
		responseData = envelope{
			"most_frequent_request": mostFrequent.Parameters,
			"hits":                  mostFrequent.Hits,
		}
	}

	// Return success response using JSON envelope format
	err = app.writeJSON(w, http.StatusOK, envelope{"data": responseData}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// validateFizzBuzzInput performs comprehensive validation on FizzBuzz input parameters
// according to the business rules and constraints defined in the acceptance criteria.
func validateFizzBuzzInput(input *data.FizzBuzzInput) *validator.Validator {
	v := validator.New()

	// Integer parameter validation
	v.Check(input.Int1 > 0, "int1", "must be a positive integer")
	v.Check(input.Int1 <= 10000, "int1", "must not be more than 10,000")
	v.Check(input.Int2 > 0, "int2", "must be a positive integer")
	v.Check(input.Int2 <= 10000, "int2", "must not be more than 10,000")
	v.Check(input.Int1 != input.Int2, "int1", "must be different from int2")
	v.Check(input.Limit > 0, "limit", "must be a positive integer")
	v.Check(input.Limit <= 100000, "limit", "must not be more than 100,000")

	// String parameter validation
	v.Check(input.Str1 != "", "str1", "must be provided")
	v.Check(len(input.Str1) <= 50, "str1", "must not be more than 50 characters")
	v.Check(input.Str2 != "", "str2", "must be provided")
	v.Check(len(input.Str2) <= 50, "str2", "must not be more than 50 characters")

	return v
}
