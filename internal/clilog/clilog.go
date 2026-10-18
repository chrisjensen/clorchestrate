// Package clilog provides a minimal timestamp-prefixed logging helper for
// CLI progress output, so long-running flows like "open" and "batch" can be
// timed phase-by-phase from their printed output.
package clilog

import (
	"fmt"
	"io"
	"time"
)

// Printf writes a UTC timestamp-prefixed, formatted line to w. UTC (rather
// than local wall-clock time) keeps these timestamps directly comparable to
// the ones scripts/worktree-checkout.sh prints on the remote server, which
// runs in a different timezone than the local machine.
func Printf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "[%s] "+format, append([]any{time.Now().UTC().Format("15:04:05.000")}, a...)...)
}
