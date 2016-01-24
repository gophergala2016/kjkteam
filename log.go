package main

import (
	"fmt"
	"os"
)

var (
	verboseLogging = false
)

// LogVerbosef prints to stdout if verbose logging is enabled
func LogVerbosef(format string, args ...interface{}) {
	if verboseLogging {
		fmt.Fprintf(os.Stdout, format, args...)
	}
}

// LogErrorf prints to stderr
func LogErrorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}
