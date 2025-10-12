package log

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-playground/assert/v2"
	log "github.com/sirupsen/logrus"
)

// Helper function to capture log output
func captureOutput(fn func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})
	fn()
	log.SetOutput(nil) // Reset to default
	return buf.String()
}

func TestDebug(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	output := captureOutput(func() {
		Debug("debug message")
	})
	assert.Equal(t, strings.Contains(output, "debug message"), true)
	assert.Equal(t, strings.Contains(output, "level=debug"), true)
}

func TestDebug_WithArgs(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	output := captureOutput(func() {
		Debug("debug with %s and %d", "string", 42)
	})
	assert.Equal(t, strings.Contains(output, "debug with string and 42"), true)
}

func TestInfo(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	output := captureOutput(func() {
		Info("info message")
	})
	assert.Equal(t, strings.Contains(output, "info message"), true)
	assert.Equal(t, strings.Contains(output, "level=info"), true)
}

func TestInfo_WithArgs(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	output := captureOutput(func() {
		Info("info with %s and %d", "string", 123)
	})
	assert.Equal(t, strings.Contains(output, "info with string and 123"), true)
}

func TestWarn(t *testing.T) {
	log.SetLevel(log.WarnLevel)
	output := captureOutput(func() {
		Warn("warning message")
	})
	assert.Equal(t, strings.Contains(output, "warning message"), true)
	assert.Equal(t, strings.Contains(output, "level=warning"), true)
}

func TestWarn_WithArgs(t *testing.T) {
	log.SetLevel(log.WarnLevel)
	output := captureOutput(func() {
		Warn("warning with %s and %d", "string", 456)
	})
	assert.Equal(t, strings.Contains(output, "warning with string and 456"), true)
}

func TestError(t *testing.T) {
	log.SetLevel(log.ErrorLevel)
	output := captureOutput(func() {
		Error("error message")
	})
	assert.Equal(t, strings.Contains(output, "error message"), true)
	assert.Equal(t, strings.Contains(output, "level=error"), true)
}

func TestError_WithArgs(t *testing.T) {
	log.SetLevel(log.ErrorLevel)
	output := captureOutput(func() {
		Error("error with %s and %d", "string", 789)
	})
	assert.Equal(t, strings.Contains(output, "error with string and 789"), true)
}

func TestMultipleArgs(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	output := captureOutput(func() {
		Info("test %s %d %v %f", "string", 42, true, 3.14)
	})
	assert.Equal(t, strings.Contains(output, "test string 42 true 3.14"), true)
}

func TestEmptyMessage(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	output := captureOutput(func() {
		Info("")
	})
	// Should not panic, should produce some output
	assert.Equal(t, len(output) > 0, true)
}

func TestSpecialCharacters(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	output := captureOutput(func() {
		Info("special chars: !@#$&*()_+-=[]{}|;':\",./<>?")
	})
	assert.Equal(t, strings.Contains(output, "special chars"), true)
}

func TestNoArgs(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	output := captureOutput(func() {
		Info("simple message with no format specifiers")
	})
	assert.Equal(t, strings.Contains(output, "simple message with no format specifiers"), true)
}

func TestLogLevels_Hierarchy(t *testing.T) {
	// Test that log level filtering works
	log.SetLevel(log.WarnLevel)

	// Debug and Info should not appear
	debugOutput := captureOutput(func() {
		Debug("debug should not appear")
	})
	assert.Equal(t, strings.Contains(debugOutput, "debug should not appear"), false)

	infoOutput := captureOutput(func() {
		Info("info should not appear")
	})
	assert.Equal(t, strings.Contains(infoOutput, "info should not appear"), false)

	// Warn and Error should appear
	warnOutput := captureOutput(func() {
		Warn("warn should appear")
	})
	assert.Equal(t, strings.Contains(warnOutput, "warn should appear"), true)

	errorOutput := captureOutput(func() {
		Error("error should appear")
	})
	assert.Equal(t, strings.Contains(errorOutput, "error should appear"), true)
}

func TestConcurrentLogging(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})

	// Logrus is thread-safe, this should not panic
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			Info("concurrent log %d", id)
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify at least some logs were written
	output := buf.String()
	assert.Equal(t, strings.Contains(output, "concurrent log"), true)
}

// Note: We cannot test Fatal() because it calls os.Exit(1) which would terminate the test process
// This is expected behavior and documented in CRITICAL_FIXES_APPLIED.md
