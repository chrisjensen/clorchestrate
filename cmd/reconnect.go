package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/chrisjensen/clorchestrate/internal/config"
	"github.com/chrisjensen/clorchestrate/internal/iterm"
	"github.com/chrisjensen/clorchestrate/internal/remoteexec"
	"github.com/spf13/cobra"
)

type screenSession struct {
	name  string
	state string
}

type configEntry struct {
	path string
	cfg  *config.Config
}

type plannedTab struct {
	sess     screenSession
	tabColor string
	runKey   string
}

func NewReconnectCmd() *cobra.Command {
	var force bool
	var restart bool
	var pause int
	cmd := &cobra.Command{
		Use:   "reconnect [config]",
		Short: "reconnect detached screen sessions",
		Long: `Reconnect detached or attached screen sessions in new iTerm2 tabs.

Without <config>: reconnect all sessions for every config in ~/.clorchestrate/.
With <config>: reconnect only sessions belonging to that config.

By default only Detached sessions are reattached (via 'screen -r'), so
sessions already open in another tab are left alone. Pass --force to
forcibly reattach any matching session (via 'screen -dr'), detaching any
currently-connected client.

Pass --restart to enumerate worktree directories derived from each config and
start a new screen session (running the configured claude command plus
'--continue') for any that does not already have one. Session names are the
worktree directory basename, except for pool/benchmark workers and their
coordinator (named "<runDirBase>_<label>" / "<runDirBase>_coordinator" so
they're unique across runs and groupable by --force).

With --restart, --pause N (default 20) stops after every N new sessions
started and waits for Enter before continuing, to avoid starting dozens of
claude processes against disk at once. --pause 0 disables pausing.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeConfigPaths,
		RunE: func(cmd *cobra.Command, args []string) error {
			if restart {
				return reconnectRestart(args, pause)
			}
			return reconnectRun(args, force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "forcibly reattach matching sessions using 'screen -dr'")
	cmd.Flags().BoolVar(&restart, "restart", false, "start new sessions for worktree dirs that have no screen session")
	cmd.Flags().IntVar(&pause, "pause", 20, "with --restart, pause for Enter after every N new sessions started (0 disables)")
	return cmd
}

func reconnectRun(args []string, force bool) error {
	var entries []configEntry
	if len(args) == 1 {
		configPath, err := resolveConfigPath(args[0])
		if err != nil {
			return err
		}
		cfg, err := config.Parse(configPath)
		if err != nil {
			return err
		}
		entries = []configEntry{{path: configPath, cfg: cfg}}
	} else {
		var err error
		entries, err = loadAllConfigs()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Fprintln(os.Stderr, "no configs found in ~/.clorchestrate/")
			return nil
		}
	}

	// Group by server to SSH once per unique server. Empty server key = local.
	byServer := map[string][]configEntry{}
	for _, e := range entries {
		byServer[e.cfg.Server] = append(byServer[e.cfg.Server], e)
	}

	for server, serverEntries := range byServer {
		sessions, err := listScreenSessions(server)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not list sessions on %s: %v\n", server, err)
			continue
		}

		for _, e := range serverEntries {
			configID, err := config.ConfigID(e.path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not determine config ID for %s: %v\n", e.path, err)
				continue
			}

			type matcher struct {
				prefix   string
				tabColor string
			}
			cfgColor := config.ResolveTabColor(e.cfg.ITermTabColor, e.path)
			matchers := []matcher{
				{prefix: configID + "_", tabColor: cfgColor},
			}
			// Also match sessions created by --restart, which are named after
			// the worktree dir basename (<worktreePrefix>-<branch>).
			// Skip when packages are defined — those matchers are built per-package below.
			if e.cfg.RemoteRepo != "" && len(e.cfg.Packages) == 0 {
				repo := strings.TrimRight(e.cfg.RemoteRepo, "/")
				if lastSlash := strings.LastIndex(repo, "/"); lastSlash >= 0 {
					wtPrefix := e.cfg.WorktreePrefix
					if wtPrefix == "" {
						wtPrefix = repo[lastSlash+1:]
					}
					matchers = append(matchers, matcher{prefix: wtPrefix + "-", tabColor: cfgColor})
				}
			}
			for _, pkg := range e.cfg.Packages {
				if pkg.Name == "" {
					continue
				}
				pkgColor := pkg.ITermTabColor
				if pkgColor == "" {
					pkgColor = e.cfg.ITermTabColor
				}
				resolvedPkgColor := config.ResolveTabColor(pkgColor, e.path)
				matchers = append(matchers, matcher{
					prefix:   pkg.Name + "_",
					tabColor: resolvedPkgColor,
				})
				// --restart sessions for packages with their own repo
				if pkg.RemoteRepo != "" {
					repo := strings.TrimRight(pkg.RemoteRepo, "/")
					if lastSlash := strings.LastIndex(repo, "/"); lastSlash >= 0 {
						wtPrefix := pkg.WorktreePrefix
						if wtPrefix == "" {
							wtPrefix = repo[lastSlash+1:]
						}
						matchers = append(matchers, matcher{prefix: wtPrefix + "-", tabColor: resolvedPkgColor})
					}
				}
			}

			screenAction := "screen -r"
			if force {
				screenAction = "screen -dr"
			}

			labels := configLabels(e.cfg)

			var planned []plannedTab
			for _, sess := range sessions {
				var matched *matcher
				for i := range matchers {
					if strings.HasPrefix(sess.name, matchers[i].prefix) {
						matched = &matchers[i]
						break
					}
				}
				if matched == nil {
					continue
				}
				if !force && sess.state != "Detached" {
					fmt.Fprintf(os.Stderr, "skipping %s (%s) — pass --force to reattach\n", sess.name, sess.state)
					continue
				}
				runKey, _ := runKeyAndLabel(sess.name, labels)
				planned = append(planned, plannedTab{sess: sess, tabColor: matched.tabColor, runKey: runKey})
			}

			// Group sibling sessions from the same run so their tabs share a
			// color and open adjacently, overriding the per-config color.
			groupOrder := make([]string, 0, len(planned))
			groups := map[string][]int{}
			for i, p := range planned {
				if _, ok := groups[p.runKey]; !ok {
					groupOrder = append(groupOrder, p.runKey)
				}
				groups[p.runKey] = append(groups[p.runKey], i)
			}
			groupOrder = mergeLegacyCoordinatorGroups(planned, groups, groupOrder)
			for _, runKey := range groupOrder {
				idxs := groups[runKey]
				if len(idxs) < 2 {
					continue
				}
				color := config.RunGroupColor(runKey)
				for _, i := range idxs {
					planned[i].tabColor = color
				}
			}
			var ordered []plannedTab
			for _, runKey := range groupOrder {
				idxs := groups[runKey]
				sort.SliceStable(idxs, func(i, j int) bool {
					return planned[idxs[i]].sess.name < planned[idxs[j]].sess.name
				})
				for _, i := range idxs {
					ordered = append(ordered, planned[i])
				}
			}

			for _, p := range ordered {
				var reconnectCmd string
				if server == "" {
					reconnectCmd = fmt.Sprintf("%s '%s'", screenAction, p.sess.name)
				} else {
					reconnectCmd = fmt.Sprintf(`ssh -t %s "%s '%s'"`, server, screenAction, p.sess.name)
				}
				if err := iterm.OpenTab(iterm.TabOptions{
					TabColorHex: p.tabColor,
					TabTitle:    p.sess.name,
					RemoteCmd:   reconnectCmd,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not open tab for session %s: %v\n", p.sess.name, err)
				}
			}
		}
	}

	return nil
}

// runKeyAndLabel strips a trailing "_coordinator" or "_<label>" (for label in
// labels) suffix from a session name, returning the shared run key and
// whether a suffix was stripped (i.e. this session is part of a multi-label
// run rather than a standalone session).
func runKeyAndLabel(name string, labels []string) (runKey string, grouped bool) {
	if stripped, ok := strings.CutSuffix(name, "_coordinator"); ok {
		return stripped, true
	}
	for _, label := range labels {
		if label == "" {
			continue
		}
		if stripped, ok := strings.CutSuffix(name, "_"+label); ok {
			return stripped, true
		}
	}
	return name, false
}

// normalizeRunKey strips a trailing "_<digits>" issue-number segment from a
// runKey. Used only as a fallback merge key for legacy coordinator sessions
// launched before a fix where the hive coordinator's session name omitted
// the run's issue number while its workers kept it (e.g. coordinator
// "foo_coordinator" vs worker "foo_123_bar"). Never used for coloring or
// primary grouping, since stripping trailing digits from every runKey would
// wrongly merge unrelated runs that share a handle but differ by issue.
func normalizeRunKey(key string) string {
	idx := strings.LastIndex(key, "_")
	if idx < 0 {
		return key
	}
	suffix := key[idx+1:]
	if suffix == "" || !allDigits(suffix) {
		return key
	}
	return key[:idx]
}

// mergeLegacyCoordinatorGroups folds an orphaned coordinator session (grouped
// alone under its own runKey) into a sibling workers' group when their
// runKeys match once the issue-number segment is stripped. This recovers
// adjacent grouping for runs started before the coordinator's session name
// was fixed to include the issue number. groups/groupOrder are mutated in
// place; the (possibly shortened) groupOrder is returned.
func mergeLegacyCoordinatorGroups(planned []plannedTab, groups map[string][]int, groupOrder []string) []string {
	for _, runKey := range groupOrder {
		idxs := groups[runKey]
		if len(idxs) != 1 || !strings.HasSuffix(planned[idxs[0]].sess.name, "_coordinator") {
			continue
		}
		for _, otherKey := range groupOrder {
			if otherKey == runKey || normalizeRunKey(otherKey) != runKey {
				continue
			}
			groups[otherKey] = append(groups[otherKey], idxs[0])
			delete(groups, runKey)
			break
		}
	}
	filtered := groupOrder[:0]
	for _, runKey := range groupOrder {
		if _, ok := groups[runKey]; ok {
			filtered = append(filtered, runKey)
		}
	}
	return filtered
}

// configLabels returns the non-empty command labels defined in cfg, in
// config order.
func configLabels(cfg *config.Config) []string {
	var labels []string
	for _, c := range cfg.Commands {
		if c.Label != "" {
			labels = append(labels, c.Label)
		}
	}
	return labels
}

func loadAllConfigs() ([]configEntry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, ".clorchestrate")
	dirEntries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	var result []configEntry
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		cfg, err := config.Parse(path)
		if err != nil {
			continue
		}
		result = append(result, configEntry{path: path, cfg: cfg})
	}
	return result, nil
}

func listScreenSessions(server string) ([]screenSession, error) {
	// screen -ls exits non-zero even when sessions exist; ignore exit code.
	out, _ := remoteexec.Run(server, "screen", "-ls").CombinedOutput()
	return parseScreenLs(string(out)), nil
}

// parseScreenLs parses the output of `screen -ls` into a list of sessions.
// Lines containing sessions look like:
//
//	\t<pid>.<name>\t(<timestamp>)\t(<state>)
//
// Some screen versions omit the timestamp field. Session names never contain
// whitespace, so we take the first field after the dot as the name and find
// the state in the last parenthesised field.
func parseScreenLs(output string) []screenSession {
	var sessions []screenSession
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		dotIdx := strings.Index(trimmed, ".")
		if dotIdx == -1 {
			continue
		}
		pid := trimmed[:dotIdx]
		if pid == "" || !allDigits(pid) {
			continue
		}
		rest := trimmed[dotIdx+1:]
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		state := ""
		for i := len(fields) - 1; i >= 1; i-- {
			f := fields[i]
			if strings.HasPrefix(f, "(") && strings.HasSuffix(f, ")") {
				state = f[1 : len(f)-1]
				break
			}
		}
		if name == "" {
			continue
		}
		sessions = append(sessions, screenSession{name: name, state: state})
	}
	return sessions
}

func allDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// reconnectRestart finds worktree directories for each config that have no
// corresponding screen session and opens a new iTerm2 tab for each, running
// claude --continue inside the worktree.
func reconnectRestart(args []string, pause int) error {
	var entries []configEntry
	if len(args) == 1 {
		configPath, err := resolveConfigPath(args[0])
		if err != nil {
			return err
		}
		cfg, err := config.Parse(configPath)
		if err != nil {
			return err
		}
		entries = []configEntry{{path: configPath, cfg: cfg}}
	} else {
		var err error
		entries, err = loadAllConfigs()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Fprintln(os.Stderr, "no configs found in ~/.clorchestrate/")
			return nil
		}
	}

	type repoTarget struct {
		remoteRepo string
		prefix     string
		tabColor   string
		cfg        *config.Config
	}

	byServer := map[string][]repoTarget{}
	for _, e := range entries {
		color := config.ResolveTabColor(e.cfg.ITermTabColor, e.path)
		var targets []repoTarget
		if len(e.cfg.Packages) == 0 {
			targets = []repoTarget{{e.cfg.RemoteRepo, e.cfg.WorktreePrefix, color, e.cfg}}
		}
		for _, pkg := range e.cfg.Packages {
			if pkg.RemoteRepo == "" {
				continue
			}
			pkgColor := pkg.ITermTabColor
			if pkgColor == "" {
				pkgColor = e.cfg.ITermTabColor
			}
			targets = append(targets, repoTarget{
				pkg.RemoteRepo,
				pkg.WorktreePrefix,
				config.ResolveTabColor(pkgColor, e.path),
				e.cfg,
			})
		}
		for _, t := range targets {
			byServer[e.cfg.Server] = append(byServer[e.cfg.Server], t)
		}
	}

	started := 0
	stdin := bufio.NewReader(os.Stdin)

	for server, targets := range byServer {
		sessionDirs, err := listSessionDirs(server)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not inspect session dirs on %s: %v\n", server, err)
			continue
		}

		for _, t := range targets {
			if t.remoteRepo == "" {
				continue
			}
			repo := strings.TrimRight(t.remoteRepo, "/")
			lastSlash := strings.LastIndex(repo, "/")
			if lastSlash < 0 {
				continue
			}
			parent := repo[:lastSlash]
			prefix := t.prefix
			if prefix == "" {
				prefix = repo[lastSlash+1:]
			}

			labels := configLabels(t.cfg)
			var discovered []discoveredWorktree
			for _, label := range labels {
				found, err := discoverWorktrees(server, parent, prefix, label)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not list %q worktrees for %s: %v\n", label, t.remoteRepo, err)
					continue
				}
				discovered = append(discovered, found...)
			}

			topDirs, err := listWorktreeDirs(server, parent, prefix)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not list worktrees for %s: %v\n", t.remoteRepo, err)
				continue
			}

			candidates := planRestartCandidates(t.cfg, labels, topDirs, discovered)

			// Sibling restarts of the same pool run (coordinator + workers)
			// share a runKey; give those a distinct, deterministic color
			// (same scheme as reconnectRun's sibling grouping) instead of the
			// config's single color, so they're visually grouped and runs
			// are distinguishable from one another. A runKey with only one
			// surviving candidate (its siblings already have live sessions)
			// isn't a group, so it keeps the config's default color.
			runKeyCounts := map[string]int{}
			for _, target := range candidates {
				if target.runKey != "" {
					runKeyCounts[target.runKey]++
				}
			}

			for _, dir := range sortedRestartDirs(candidates) {
				if sessionDirs[dir] {
					fmt.Fprintf(os.Stderr, "skipping %s — already open in a screen session\n", dir)
					continue
				}
				target := candidates[dir]
				// Start screen with a login shell cd'd into the worktree, then
				// type the resolved claude command + --continue as a followup
				// (same pattern as open's noClaude path + buildFollowupCmd, so
				// the login profile is sourced before claude runs). Assumes
				// every configured command accepts a trailing --continue flag
				// the same way the default command does.
				var remoteCmd string
				if server == "" {
					remoteCmd = fmt.Sprintf("screen -S %s bash -c 'cd %s && exec bash -l'", target.sessionName, dir)
				} else {
					remoteCmd = fmt.Sprintf(`ssh -t %s "screen -S %s bash -c 'cd %s && exec bash -l'"`, server, target.sessionName, dir)
				}
				tabColor := t.tabColor
				if target.runKey != "" && runKeyCounts[target.runKey] >= 2 {
					tabColor = config.RunGroupColor(target.runKey)
				}
				fmt.Fprintf(os.Stderr, "starting session %q for %s\n", target.sessionName, dir)
				if err := iterm.OpenTab(iterm.TabOptions{
					TabColorHex: tabColor,
					TabTitle:    target.sessionName,
					RemoteCmd:   remoteCmd,
					FollowupCmd: target.cmd + " --continue",
					LocalMode:   server == "",
				}); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not open tab for %s: %v\n", dir, err)
				}

				started++
				if pause > 0 && started%pause == 0 {
					fmt.Fprintf(os.Stderr, "started %d sessions — press Enter to continue...\n", started)
					stdin.ReadString('\n')
				}
			}
		}
	}

	return nil
}

// restartTarget is the resolved command, screen session name, and
// run-grouping key for one worktree directory restart will start a session
// in. runKey is empty for worktrees that aren't part of a multi-session
// pool/benchmark run.
type restartTarget struct {
	cmd         string
	sessionName string
	runKey      string
}

// planRestartCandidates resolves the claude command, screen session name,
// and run-grouping key to use for every worktree directory a restart should
// consider, covering three cases that mirror how launchLabelSet
// (cmd/open_labelset.go) assigns commands when a session is first launched:
//
//   - a labeled worktree (legacy sibling <prefix>-<base>-<label>, or pool
//     child <rundir>/<label>) uses cfg.CommandByLabel(label), grouped under
//     its discoveredWorktree.taskKey. Its session name is the dir basename
//     for the sibling layout (already unique: "<prefix>-<base>-<label>"), or
//     "<runDirBase>_<label>" for the pool child layout — the bare label
//     alone is not unique across different runs and matches none of
//     reconnectRun's prefix matchers, so --force could never find or color
//     it.
//   - a pool run dir's root (the coordinator, found because one of its
//     children is a discovered label dir) uses cfg.CommandByLabel(labels[0]),
//     mirroring coordOpts.commandLabel = labels[0], grouped under the same
//     taskKey as its workers, with session name "<runDirBase>_coordinator".
//   - any other top-level worktree dir (a plain, unlabeled worktree) uses
//     cfg.DefaultClaudeCmd(), its own dir basename as session name, and has
//     no runKey (ungrouped).
//
// discovered is the concatenation of discoverWorktrees results for every
// label; topDirs is the flat listWorktreeDirs glob result.
func planRestartCandidates(cfg *config.Config, labels []string, topDirs []string, discovered []discoveredWorktree) map[string]restartTarget {
	candidates := map[string]restartTarget{}
	poolRunDirs := map[string]string{} // runDir -> taskKey

	resolveLabel := func(label string) string {
		cmd, err := cfg.CommandByLabel(label)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v; using default command\n", err)
			return cfg.DefaultClaudeCmd()
		}
		return cmd
	}

	for _, w := range discovered {
		label := labelOf(w, labels)
		sessionName := filepath.Base(w.dir)
		if w.runDir != "" {
			sessionName = filepath.Base(w.runDir) + "_" + label
		}
		candidates[w.dir] = restartTarget{cmd: resolveLabel(label), sessionName: sessionName, runKey: w.taskKey}
		if w.runDir != "" {
			poolRunDirs[w.runDir] = w.taskKey
		}
	}

	for _, dir := range topDirs {
		if _, ok := candidates[dir]; ok {
			continue // already resolved above as a legacy sibling worktree
		}
		if taskKey, ok := poolRunDirs[dir]; ok {
			cmd := cfg.DefaultClaudeCmd()
			if len(labels) > 0 {
				cmd = resolveLabel(labels[0])
			}
			candidates[dir] = restartTarget{cmd: cmd, sessionName: filepath.Base(dir) + "_coordinator", runKey: taskKey}
			continue
		}
		candidates[dir] = restartTarget{cmd: cfg.DefaultClaudeCmd(), sessionName: filepath.Base(dir)}
	}

	return candidates
}

// labelOf returns the label a discoveredWorktree was found under: its
// directory basename for the pool child layout, or the basename's trailing
// "-<label>" suffix for the legacy sibling layout.
func labelOf(w discoveredWorktree, labels []string) string {
	if w.runDir != "" {
		return filepath.Base(w.dir)
	}
	base := filepath.Base(w.dir)
	for _, label := range labels {
		if strings.HasSuffix(base, "-"+label) {
			return label
		}
	}
	return ""
}

func sortedRestartDirs(m map[string]restartTarget) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// listSessionDirs returns the set of working directories currently held by
// child processes of all screen sessions. This detects sessions regardless of
// their name, covering both sessions created by `open` (configID_handle) and
// sessions created by --restart (dir basename).
//
// On Linux the cwd is read from /proc/<pid>/cwd; on macOS via lsof. The shell
// command tries /proc first and falls back to lsof so it works on both.
func listSessionDirs(server string) (map[string]bool, error) {
	shellCmd := `screen -ls 2>/dev/null | grep -oE '[0-9]+\.' | tr -d '.' | while read pid; do` +
		` pgrep -P "$pid" 2>/dev/null | while read child; do` +
		` readlink /proc/$child/cwd 2>/dev/null ||` +
		` lsof -a -d cwd -p $child -Fn 2>/dev/null | grep '^n' | sed 's/^n//';` +
		` done; done 2>/dev/null`
	out, _ := remoteexec.RunShell(server, shellCmd).Output()
	dirs := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			dirs[line] = true
		}
	}
	return dirs, nil
}

// listWorktreeDirs returns directories under parent whose names start with
// prefix followed by "-". Runs locally when server is "".
func listWorktreeDirs(server, parent, prefix string) ([]string, error) {
	pattern := parent + "/" + prefix + "-*"
	shellCmd := fmt.Sprintf("ls -d %s 2>/dev/null", pattern)
	out, _ := remoteexec.RunShell(server, shellCmd).Output()
	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			dirs = append(dirs, line)
		}
	}
	return dirs, nil
}
