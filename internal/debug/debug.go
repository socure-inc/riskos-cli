// Package debug provides a zero-cost debug logger.
// When disabled (default), Log calls are no-ops. Call Enable() once at startup
// to activate output to stderr.
package debug

import (
	"fmt"
	"os"
)

// Logger is the interface for debug output.
type Logger interface {
	Log(format string, args ...any)
}

type nop struct{}

func (nop) Log(string, ...any) {}

type stderrLogger struct{}

func (stderrLogger) Log(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
}

var logger Logger = nop{}

// Enable activates debug logging to stderr.
func Enable() { logger = stderrLogger{} }

// Log writes a formatted debug message. No-op when debug is disabled.
func Log(format string, args ...any) { logger.Log(format, args...) }
