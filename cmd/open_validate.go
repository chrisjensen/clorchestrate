package cmd

import (
	"fmt"
	"regexp"
)

// safeNameRE is the charset allowed in user-supplied handles and branch
// names. These values are interpolated into ssh/screen command strings
// (session names, /tmp/task-<slug>.* paths, worktree-checkout.sh args)
// without shell escaping, so anything outside [A-Za-z0-9._-] is rejected at
// entry. benchmarkLabel/commandLabel need no validation: they are always
// config-defined [[command]] labels resolved via ResolveCommand.
var safeNameRE = regexp.MustCompile(`^[A-Za-z0-9._-]*$`)

// SafeName is a handle or branch value that has passed NewSafeName's
// charset check, so it's safe to splice into an unescaped ssh/screen
// command string. Composite values built by concatenating SafeNames with
// literal separators ("_", "-coordinator", etc.) are safe by construction
// and are carried as plain strings once built — the type exists to stop an
// unvalidated string reaching a Sprintf, not to track every downstream
// composite.
type SafeName string

// String returns the underlying value, for use where a SafeName is
// concatenated into a composite session name or slug.
func (s SafeName) String() string { return string(s) }

// NewSafeName rejects kind ("handle"/"branch") values containing characters
// outside [A-Za-z0-9._-]. Empty is allowed (both are optional).
func NewSafeName(kind, value string) (SafeName, error) {
	if safeNameRE.MatchString(value) {
		return SafeName(value), nil
	}
	return "", fmt.Errorf("invalid %s %q: only letters, digits, '.', '_' and '-' allowed (no spaces or shell characters)", kind, value)
}
