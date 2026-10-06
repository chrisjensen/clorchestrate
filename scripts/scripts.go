package scripts

import _ "embed"

//go:embed worktree-checkout.sh
var WorktreeCheckout []byte

//go:embed pool-signal.sh
var PoolSignal []byte

type Script struct {
	Name    string
	Content []byte
	// Dir is the target directory on the server ("" means ~/bin). pool-signal.sh
	// goes to ~/.local/bin so pool skills can invoke it bare (that dir is on the
	// login-shell PATH per the project's server PATH requirements).
	Dir string
}

func All() []Script {
	return []Script{
		{Name: "worktree-checkout.sh", Content: WorktreeCheckout},
		{Name: "pool-signal.sh", Content: PoolSignal, Dir: "~/.local/bin"},
	}
}
