package cmd

import (
	"testing"

	"github.com/chrisjensen/clorchestrate/internal/config"
)

func TestRunKeyAndLabel(t *testing.T) {
	labels := []string{"a", "b"}

	cases := []struct {
		name        string
		wantRunKey  string
		wantGrouped bool
	}{
		{"myconfig_task123_a", "myconfig_task123", true},
		{"myconfig_task123_b", "myconfig_task123", true},
		{"myconfig_task123_coordinator", "myconfig_task123", true},
		{"myconfig_task123", "myconfig_task123", false},
		{"myconfig_task123_ab", "myconfig_task123_ab", false},
	}
	for _, c := range cases {
		runKey, grouped := runKeyAndLabel(c.name, labels)
		if runKey != c.wantRunKey || grouped != c.wantGrouped {
			t.Errorf("runKeyAndLabel(%q) = (%q, %v), want (%q, %v)", c.name, runKey, grouped, c.wantRunKey, c.wantGrouped)
		}
	}
}

// TestMergeLegacyCoordinatorGroups covers a coordinator launched before its
// session name was fixed to include the run's issue number: it groups alone
// under a runKey lacking the issue segment its workers have, so it must be
// folded into their group instead of opening in a separate, uncolored tab.
func TestMergeLegacyCoordinatorGroups(t *testing.T) {
	labels := []string{"zai"}
	planned := []plannedTab{
		{sess: screenSession{name: "extractor_internal-links-missing_10720_zai"}},
		{sess: screenSession{name: "extractor_internal-links-missing_coordinator"}},
	}
	for i := range planned {
		planned[i].runKey, _ = runKeyAndLabel(planned[i].sess.name, labels)
	}

	groupOrder := make([]string, 0, len(planned))
	groups := map[string][]int{}
	for i, p := range planned {
		if _, ok := groups[p.runKey]; !ok {
			groupOrder = append(groupOrder, p.runKey)
		}
		groups[p.runKey] = append(groups[p.runKey], i)
	}

	groupOrder = mergeLegacyCoordinatorGroups(planned, groups, groupOrder)

	if len(groupOrder) != 1 {
		t.Fatalf("groupOrder = %v, want a single merged group", groupOrder)
	}
	idxs := groups[groupOrder[0]]
	if len(idxs) != 2 {
		t.Fatalf("merged group has %d members, want 2", len(idxs))
	}
}

// TestMergeLegacyCoordinatorGroups_NoFalseMerge ensures distinct runs (same
// handle, different issue numbers) are never merged just because one of
// their runKeys happens to end in a plain numeric segment.
func TestMergeLegacyCoordinatorGroups_NoFalseMerge(t *testing.T) {
	planned := []plannedTab{
		{runKey: "myconfig_task_111"},
		{runKey: "myconfig_task_222"},
	}
	groupOrder := []string{"myconfig_task_111", "myconfig_task_222"}
	groups := map[string][]int{
		"myconfig_task_111": {0},
		"myconfig_task_222": {1},
	}

	got := mergeLegacyCoordinatorGroups(planned, groups, groupOrder)

	if len(got) != 2 {
		t.Errorf("groupOrder = %v, want no merge (no coordinator session present)", got)
	}
}

func abZaiConfig() *config.Config {
	return &config.Config{
		Commands: []config.Command{
			{Label: "claude", Cmd: "claude"},
			{Label: "zai", Cmd: "zai"},
		},
	}
}

func TestPlanRestartCandidates_PoolLayout(t *testing.T) {
	cfg := abZaiConfig()
	labels := configLabels(cfg)
	topDirs := []string{"/repo/myrepo-feature"}
	discovered := []discoveredWorktree{
		{dir: "/repo/myrepo-feature/claude", taskKey: "myrepo-feature", runDir: "/repo/myrepo-feature"},
		{dir: "/repo/myrepo-feature/zai", taskKey: "myrepo-feature", runDir: "/repo/myrepo-feature"},
	}

	got := planRestartCandidates(cfg, labels, topDirs, discovered)

	want := map[string]restartTarget{
		"/repo/myrepo-feature/claude": {cmd: "claude", sessionName: "myrepo-feature_claude", runKey: "myrepo-feature"},
		"/repo/myrepo-feature/zai":    {cmd: "zai", sessionName: "myrepo-feature_zai", runKey: "myrepo-feature"},
		"/repo/myrepo-feature":        {cmd: "claude", sessionName: "myrepo-feature_coordinator", runKey: "myrepo-feature"}, // coordinator uses labels[0]
	}
	if len(got) != len(want) {
		t.Fatalf("planRestartCandidates() = %v, want %v", got, want)
	}
	for dir, wantTarget := range want {
		if got[dir] != wantTarget {
			t.Errorf("candidates[%q] = %+v, want %+v", dir, got[dir], wantTarget)
		}
	}
}

func TestPlanRestartCandidates_PlainWorktree(t *testing.T) {
	cfg := abZaiConfig()
	labels := configLabels(cfg)
	topDirs := []string{"/repo/myrepo-plain"}

	got := planRestartCandidates(cfg, labels, topDirs, nil)

	want := restartTarget{cmd: cfg.DefaultClaudeCmd(), sessionName: "myrepo-plain"}
	if len(got) != 1 || got["/repo/myrepo-plain"] != want {
		t.Errorf("planRestartCandidates() = %v, want %v", got, want)
	}
}

func TestPlanRestartCandidates_LegacySiblingLayout(t *testing.T) {
	cfg := abZaiConfig()
	labels := configLabels(cfg)
	topDirs := []string{"/repo/myrepo-feature-zai"}
	discovered := []discoveredWorktree{
		{dir: "/repo/myrepo-feature-zai", taskKey: "myrepo-feature"},
	}

	got := planRestartCandidates(cfg, labels, topDirs, discovered)

	if len(got) != 1 {
		t.Fatalf("planRestartCandidates() = %v, want exactly one candidate (no duplicate)", got)
	}
	want := restartTarget{cmd: "zai", sessionName: "myrepo-feature-zai", runKey: "myrepo-feature"}
	if got["/repo/myrepo-feature-zai"] != want {
		t.Errorf("candidates[%q] = %+v, want %+v", "/repo/myrepo-feature-zai", got["/repo/myrepo-feature-zai"], want)
	}
}
