package adapters

import (
	"testing"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

func TestNewSentryErrorReporter_WithEmptyDSN(t *testing.T) {
	assert := test.NewAssertions(t)

	// Create reporter with empty DSN (safe for testing)
	reporter := NewSentryErrorReporter("", "test")

	assert.NotNil(reporter)
	// Verify it implements the interface
	var _ f.ErrorReporter = reporter
}

func TestNewSentryErrorReporter_WithInvalidDSN(t *testing.T) {
	assert := test.NewAssertions(t)

	// Create reporter with invalid DSN
	// This should not panic, just log initialization error
	reporter := NewSentryErrorReporter("invalid-dsn", "test")

	assert.NotNil(reporter)
}

func TestNewSentryErrorReporter_WithDifferentEnvironments(t *testing.T) {
	assert := test.NewAssertions(t)

	// Test with different environment names
	envs := []string{"development", "staging", "production", "test", ""}

	for _, env := range envs {
		reporter := NewSentryErrorReporter("", env)
		assert.NotNil(reporter)
	}
}

func TestSentryErrorReporter_InterfaceCompliance(t *testing.T) {
	assert := test.NewAssertions(t)

	reporter := NewSentryErrorReporter("", "test")

	// Verify it implements ErrorReporter interface
	var _ f.ErrorReporter = reporter

	// Verify the client is set (even with empty DSN)
	sentryReporter := reporter.(*SentryErrorReporter)
	assert.NotNil(sentryReporter.client)
}

// NOTE: Testing actual error reporting would require:
// 1. A valid Sentry DSN
// 2. Mocking the Sentry client
// 3. Checking that events are captured
//
// For Tier 1 testing, we focus on:
// - Constructor works with various inputs
// - Interface compliance
// - No panics on initialization
//
// Full error reporting functionality would be tested in integration tests
// with a real or mocked Sentry backend.
