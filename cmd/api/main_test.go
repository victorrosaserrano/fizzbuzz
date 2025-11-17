package main

import (
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// Test that version variables exist and can be set
	// These should be set at build time via ldflags
	t.Run("version and buildTime variables exist and are addressable", func(t *testing.T) {
		// Variables should be declared and addressable (not pointers themselves)
		versionPtr := &version
		buildTimePtr := &buildTime

		if versionPtr == nil {
			t.Error("version variable should be addressable")
		}
		if buildTimePtr == nil {
			t.Error("buildTime variable should be addressable")
		}
	})

	t.Run("version variables can be set", func(t *testing.T) {
		// Test that variables can be modified (proving they're not constants)
		originalVersion := version
		originalBuildTime := buildTime

		// Temporarily set values
		version = "test-version"
		buildTime = "test-buildtime"

		if version != "test-version" {
			t.Error("version variable should be settable")
		}
		if buildTime != "test-buildtime" {
			t.Error("buildTime variable should be settable")
		}

		// Restore original values
		version = originalVersion
		buildTime = originalBuildTime
	})
}
