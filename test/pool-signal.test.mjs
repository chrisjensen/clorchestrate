// Unit tests for scripts/pool-signal.sh: `wait` blocks until every label signals (and
// fails loud on an empty run dir), `wait-one` rejects non-signal files, `emit impl`
// requires a commit since plan.merged, and `durations` reads sentinel mtimes. (Uses
// the poll fallback when inotifywait is absent.)

import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdtempSync, mkdirSync, rmSync, writeFileSync, utimesSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';

const BIN = resolve(dirname(fileURLToPath(import.meta.url)), '..', 'scripts', 'pool-signal.sh');

// Labels are the run dir's subdirs (each a worker worktree named for its label).
function coordWithWorkers(labels) {
  const dir = mkdtempSync(join(tmpdir(), 'pool-'));
  for (const label of labels) mkdirSync(join(dir, label));
  return dir;
}

const sh = (...args) => spawnSync(BIN, args, { encoding: 'utf8' });

describe('pool-signal.sh', () => {
  test('wait blocks until every label signals, then exits 0', async () => {
    const dir = coordWithWorkers(['claude', 'zclaude']);
    try {
      const waiter = spawn(BIN, ['wait', dir, 'plan'], { encoding: 'utf8' });
      const exited = new Promise((res) => waiter.on('exit', (code) => res(code)));

      sh('emit', dir, 'claude', 'plan');
      await delay(300);
      assert.equal(waiter.exitCode, null, 'wait should still block with one label outstanding');

      sh('emit', dir, 'zclaude', 'plan');
      const code = await exited;
      assert.equal(code, 0);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test('wait fails loud on an empty run dir', () => {
    const dir = mkdtempSync(join(tmpdir(), 'pool-'));
    try {
      const r = sh('wait', dir, 'plan');
      assert.equal(r.status, 2);
      assert.match(r.stderr, /no worker subdirs/);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test('durations computes plan/impl seconds from sentinel mtimes', () => {
    const dir = coordWithWorkers(['claude']);
    try {
      // Touch each sentinel with a controlled mtime, 10s apart, in stage order.
      const touchAt = (path, secondsFromEpoch) => {
        writeFileSync(path, '');
        utimesSync(path, secondsFromEpoch, secondsFromEpoch);
      };
      const t0 = Math.floor(Date.now() / 1000) - 1000;
      touchAt(join(dir, 'task.md'), t0);
      touchAt(join(dir, 'claude.plan.done'), t0 + 10);
      touchAt(join(dir, 'plan.merged'), t0 + 20);
      touchAt(join(dir, 'claude.impl.done'), t0 + 45);

      const r = sh('durations', dir, 'claude');
      assert.equal(r.status, 0, r.stderr);
      assert.deepEqual(JSON.parse(r.stdout), { planDurationSeconds: 10, implDurationSeconds: 25 });
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test('wait-one rejects a plan content file instead of a signal', () => {
    const dir = mkdtempSync(join(tmpdir(), 'pool-'));
    try {
      const r = sh('wait-one', dir, 'PLAN.hybrid.md');
      assert.equal(r.status, 2);
      assert.match(r.stderr, /not a signal file/);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test('emit impl refuses until the branch has a commit since plan.merged', () => {
    const dir = coordWithWorkers(['claude']);
    const worktree = join(dir, 'claude');
    const mergedAt = Math.floor(Date.now() / 1000) - 100;
    const commitAt = (secondsFromEpoch, message) => {
      const date = `@${secondsFromEpoch} +0000`;
      const r = spawnSync(
        'git',
        ['-C', worktree, '-c', 'user.name=t', '-c', 'user.email=t@t', 'commit', '-q', '--allow-empty', '-m', message],
        { encoding: 'utf8', env: { ...process.env, GIT_AUTHOR_DATE: date, GIT_COMMITTER_DATE: date } },
      );
      assert.equal(r.status, 0, r.stderr);
    };
    try {
      assert.equal(spawnSync('git', ['-C', worktree, 'init', '-q']).status, 0);
      commitAt(mergedAt - 50, 'plan-stage commit');
      writeFileSync(join(dir, 'plan.merged'), '');
      utimesSync(join(dir, 'plan.merged'), mergedAt, mergedAt);

      const refused = sh('emit', dir, 'claude', 'impl');
      assert.equal(refused.status, 2);
      assert.match(refused.stderr, /no commits .* since plan\.merged/);
      assert.equal(existsSync(join(dir, 'claude.impl.done')), false);

      commitAt(mergedAt + 10, 'implementation');
      const emitted = sh('emit', dir, 'claude', 'impl');
      assert.equal(emitted.status, 0, emitted.stderr);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test('durations fails loud when a sentinel is missing', () => {
    const dir = coordWithWorkers(['claude']);
    try {
      const r = sh('durations', dir, 'claude');
      assert.equal(r.status, 2);
      assert.match(r.stderr, /missing/);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});
