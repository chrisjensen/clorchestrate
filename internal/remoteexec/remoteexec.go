// Package remoteexec runs a command locally or over ssh depending on
// whether a server is set, so callers don't each re-implement the
// sh -c/ssh branch and its quoting.
package remoteexec

import (
	"os/exec"
	"strings"

	"github.com/chrisjensen/clorchestrate/internal/conffile"
)

// Run executes argv with no shell involved: locally as exec.Command(argv[0],
// argv[1:]...), or over ssh as a single shell-quoted argument (ssh always
// hands its command to a remote shell, so quoting there is unavoidable, but
// none of argv's elements are interpreted as shell syntax by it either way).
// Use this whenever the command doesn't need pipes, redirection, or other
// shell syntax.
func Run(server string, argv ...string) *exec.Cmd {
	if server == "" {
		return exec.Command(argv[0], argv[1:]...)
	}
	return exec.Command("ssh", server, Quote(argv))
}

// RunShell executes shellFragment through a shell: locally via `sh -c`, or
// over ssh. shellFragment must already be safe to splice into a shell
// command — callers build it from remoteexec.Quote / SafeName-validated
// parts / conffile.ShellQuote, not from unvalidated input.
func RunShell(server, shellFragment string) *exec.Cmd {
	if server == "" {
		return exec.Command("sh", "-c", shellFragment)
	}
	return exec.Command("ssh", server, shellFragment)
}

// Quote joins argv into a single shell-safe string, quoting each element.
func Quote(argv []string) string {
	quoted := make([]string, len(argv))
	for i, a := range argv {
		quoted[i] = conffile.ShellQuote(a)
	}
	return strings.Join(quoted, " ")
}
