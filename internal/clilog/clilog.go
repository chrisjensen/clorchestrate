// Package clilog provides a minimal timestamp-prefixed logging helper for
// CLI progress output, so long-running flows like "open" and "batch" can be
// timed phase-by-phase from their printed output.
package clilog

import (
	"fmt"
	"io"
	"time"
)

// Printf writes a timestamp-prefixed, formatted line to w.
func Printf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "[%s] "+format, append([]any{time.Now().Format("15:04:05.000")}, a...)...)
}
