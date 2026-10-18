package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chrisjensen/clorchestrate/internal/clilog"
	"github.com/chrisjensen/clorchestrate/internal/config"
)

// launchLabelSet opens one session per label: a plain labeled session for a
// single label, or — when there are 2+ labels — pool mode (a shared run dir,
// worker worktrees at <runDir>/<label>, plus a coordinator session). This is
// the one implementation shared by `open --benchmark` and batch's per-task
// `command:`/`--benchmark` handling, so both behave identically for the same
// label count. resolveBranch returns the branch to use for a given label
// (batch creates/reuses it via gh; open's own --benchmark just suffixes the
// user-supplied branch). baseRef, when non-empty, is the branch to check
// "commits ahead" against (batch's resolved default base branch) — passing
// "" skips that check, matching open's own --benchmark behavior.
func launchLabelSet(configPath, handle, baseBranch, baseRef string, cfg *config.Config, labels []string, baseOpts openOptions, resolveBranch func(label string) (string, error)) error {
	pool := len(labels) >= 2
	runDir := worktreePath(cfg.RemoteRepo, baseBranch, cfg.WorktreePrefix)

	// Branch resolution (gh API calls — create or look up) doesn't touch the
	// local git repo, so it's safe to run concurrently across labels even
	// though the checkout loop below must stay sequential (worktree add
	// writes to the shared .git/config).
	branches, err := resolveBranchesConcurrently(labels, resolveBranch)
	if err != nil {
		return err
	}

	for i, label := range labels {
		branch := branches[i]

		worktreeDir := worktreePath(cfg.RemoteRepo, branch, cfg.WorktreePrefix)
		if pool {
			worktreeDir = filepath.Join(runDir, label)
		}

		opts := baseOpts
		opts.benchmarkLabel = label
		if baseRef != "" {
			ahead, err := branchAheadCount(cfg.Server, worktreeDir, baseRef)
			if err != nil {
				clilog.Printf(os.Stderr, "  warning: could not check commits ahead for %s: %v — proceeding with planning session\n", branch, err)
				ahead = 0
			}
			if ahead > 0 {
				clilog.Printf(os.Stderr, "  Branch is %d commit(s) ahead of %s — opening shell in worktree (no Claude)\n", ahead, baseRef)
			}
			opts.noClaude = ahead > 0
		}
		if pool {
			opts.poolRole = poolRoleWorker
			opts.runDir = runDir
		}
		if err := openRun(configPath, handle, branch, opts); err != nil {
			return fmt.Errorf("open for %s/%s: %w", handle, label, err)
		}
	}

	if pool {
		// The coordinator has no branch of its own, but keeps baseOpts.issue
		// (the run's shared issue number) so its runKey/tab color matches its
		// workers'; openRun special-cases poolRole==coordinator to skip the
		// branch-vs-issue argument validation that issue-without-branch would
		// otherwise fail.
		coordOpts := baseOpts
		coordOpts.benchmarkLabel = ""
		coordOpts.poolRole = poolRoleCoordinator
		coordOpts.runDir = runDir
		coordOpts.commandLabel = labels[0]
		if err := openRun(configPath, handle, "", coordOpts); err != nil {
			return fmt.Errorf("open coordinator for %s: %w", handle, err)
		}
	}
	return nil
}

// worktreePath mirrors the layout used by scripts/worktree-checkout.sh:
// $(dirname REMOTE_REPO)/<prefix>-<sanitized-branch>. prefix defaults to the
// repo basename when empty. The server will expand ~ when the command runs.
func worktreePath(remoteRepo, branch, prefix string) string {
	repo := strings.TrimRight(remoteRepo, "/")
	parent := "."
	lastSlash := strings.LastIndex(repo, "/")
	if lastSlash > 0 {
		parent = repo[:lastSlash]
	} else if lastSlash == 0 {
		parent = "/"
	}
	if prefix == "" {
		prefix = repo[lastSlash+1:]
	}
	sanitized := strings.ReplaceAll(branch, "/", "-")
	return parent + "/" + prefix + "-" + sanitized
}

// resolveBranchesConcurrently runs resolveBranch for every label in parallel,
// preserving label order in the returned slice. Returns the first error
// encountered (if any); order of errors across labels is not meaningful
// since they race.
func resolveBranchesConcurrently(labels []string, resolveBranch func(label string) (string, error)) ([]string, error) {
	branches := make([]string, len(labels))
	errs := make([]error, len(labels))
	var wg sync.WaitGroup
	for i, label := range labels {
		wg.Add(1)
		go func(i int, label string) {
			defer wg.Done()
			branches[i], errs[i] = resolveBranch(label)
		}(i, label)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return branches, nil
}
