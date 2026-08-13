#!/usr/bin/env sh
set -eu

# worktree-checkout: replace tokensave seeding with serena
git add scripts/worktree-checkout.sh
git commit -F- <<'EOF'
worktree-checkout: replace tokensave with serena

The tokensave seeding block (fetch/checkout/pull/stash/sync/copy) is
replaced with a simpler rsync of .serena/ from the main worktree,
excluding cache/ (the LSP index rebuilds automatically). project_name
in project.local.yml is updated to match the new worktree directory name.

git fetch is scoped to the target branch only rather than fetching all
refs, reducing unnecessary network traffic on large remotes.

Resume detection now reports "serena: present/not found" instead of
the tokensave equivalent.
EOF

# review: screen-based evaluation with caching, retry, and improved output parsing
git add cmd/review.go
git commit -F- <<'EOF'
review: evaluate benchmark implementations across command variants

Command description updated to reflect that the primary purpose is
comparing implementation quality across command variants, not reporting
token usage (which is supplementary and not fully implemented for all
agents).

Evaluation agents run in detached screen sessions (screen -dmS) rather
than blocking SSH calls. Output is written to a result file
(/tmp/clorchestrate-eval-<worktree>.out) so that a dropped connection
does not lose work. Local evaluations use nohup for the same benefit.

On re-run, a cached result file is reused; an in-progress screen session
is polled until it exits; a new session is started only if neither exists.
--fresh deletes result files and kills any running evaluation sessions
before starting, and implies --evaluate.

runEvaluation retries up to 3 times with a 60-second delay when the
result file contains a 529-overloaded error. deleteEvalArtifacts clears
artifacts between retries.

JSON extraction switches from strings.LastIndex to a brace-depth walker
(findJSONBounds) so prose containing braces does not corrupt the parse.
Prose before a code fence is now captured alongside prose after it.

Completeness scoring is per-row rather than cross-group: each row's
score = (all_aspects_count - missing_count) * 100 / all_aspects_count.

review now scans all packages within a config (not just the base config),
deduplicating worktree dirs that appear under multiple package prefixes.
Only worktrees with a .clorchestrate-done file are included.

homeCache avoids repeated SSH round-trips for the same server when
iterating over multiple configs.
EOF

# prompts: extract inline prompt strings to embedded templates; fix benchmark prompt
git add prompts/ cmd/open.go cmd/open_test.go
git commit -F- <<'EOF'
prompts: move inline prompts to embedded templates; fix benchmark prompt

buildPrompt and buildTaskPrompt are removed from cmd/open.go. Their
content moves to text/template files under prompts/ (prompt.txt,
eval.txt), embedded at build time via //go:embed, mirroring the
scripts/ pattern. Callers use prompts.Prompt(PromptData{...}).

Benchmark sessions now always build a prompt (including planning_context)
even when no --extra-context is provided — previously the prompt was
skipped entirely, so planning_context was never sent. The ## Benchmark
instructions heading and duplicate worktree line are removed; the
template already includes the worktree instruction for all modes. The
two .clorchestrate-done lines are appended without a heading.

The prompt template makes the intro line and ## Task description section
conditional on Description being non-empty, so a benchmark-only prompt
renders cleanly with just the repo context and planning instructions.

buildFollowupCmd supports a {prompt} placeholder in configured claude
commands: if the command string contains {prompt}, the prompt expression
is substituted in place rather than appended as a trailing argument.
EOF

# config, flock, reconnect: require explicit package selection; planning_context per package
git add internal/config/config.go cmd/flock.go cmd/reconnect.go
git commit -F- <<'EOF'
config, flock, reconnect: require explicit package selection

ResolvePackage now returns an error when called with an empty name on a
config that defines [[package]] blocks, instead of silently falling
through to the top-level config. Configs with no packages are unchanged.

Package-level fields on the resolved config are cleared before overlaying
the selected package, so global values do not silently bleed through when
a package omits a field.

PlanningContext is added to Package so it can be overridden per package.
Config gains a Name field; ConfigID checks it before the filename-stem
algorithm, giving configs with name = "..." a stable identifier.

flock: tasks in a config with packages must include a package: key;
tasks that omit it are skipped with an error message.

reconnect: worktree matchers and restart targets are skipped for the
top-level config when packages are defined — per-package targets are
built from the [[package]] list instead, avoiding duplicate matches.
EOF

# commit.sh: record the above
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record serena, review evaluation, prompts, and package-selection commits
EOF
