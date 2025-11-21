// Package validator provides a custom input validation framework for the FizzBuzz API.
// Implements field-specific error collection and validation helpers following "Let's Go Further" patterns.
package validator

import (
	"regexp"
	"slices"
)

// Validator provides thread-safe field-specific error collection for input validation.
// Designed for single-request validation (not concurrent across requests).
type Validator struct {
	// errors stores field-specific validation error messages
	errors map[string]string
}

// New creates and returns a new Validator instance with empty error state.
func New() *Validator {
	return &Validator{
		errors: make(map[string]string),
	}
}

// Valid returns true if the validator contains no validation errors.
func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

// AddError adds a field-specific error message to the validator.
// If an error already exists for the field, it will be overwritten.
func (v *Validator) AddError(key, message string) {
	if v.errors == nil {
		v.errors = make(map[string]string)
	}
	v.errors[key] = message
}

// Check evaluates a condition and adds an error message if the condition is false.
// This is the primary method for performing validation checks.
func (v *Validator) Check(condition bool, key, message string) {
	if !condition {
		v.AddError(key, message)
	}
}

// ErrorMap returns a copy of the errors map for safe external access.
// The returned map is a copy to prevent external modification of internal state.
func (v *Validator) ErrorMap() map[string]string {
	if len(v.errors) == 0 {
		return nil
	}

	errorsCopy := make(map[string]string, len(v.errors))
	for key, message := range v.errors {
		errorsCopy[key] = message
	}
	return errorsCopy
}

// Clear resets the validator to an empty error state for reuse.
// Useful when reusing validator instances across multiple validation passes.
func (v *Validator) Clear() {
	v.errors = make(map[string]string)
}

// PermittedValue returns true if value is contained in the list of permitted values.
// Uses generics for type safety and supports any comparable type.
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

// Matches returns true if the value matches the provided regular expression pattern.
// Returns false if the pattern is nil or if the value doesn't match.
func Matches(value string, pattern *regexp.Regexp) bool {
	if pattern == nil {
		return false
	}
	return pattern.MatchString(value)
}

// Unique returns true if all values in the slice are unique (no duplicates).
// Uses generics for type safety and supports any comparable type.
func Unique[T comparable](values []T) bool {
	seen := make(map[T]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

// In returns true if value is found in the provided list.
// This is an alias for PermittedValue with slice parameter for convenience.
func In[T comparable](value T, list []T) bool {
	return slices.Contains(list, value)
}
