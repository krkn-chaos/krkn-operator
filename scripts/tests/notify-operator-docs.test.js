'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execFileSync, spawnSync } = require('node:child_process');
const test = require('node:test');

const repoRoot = path.resolve(__dirname, '../..');
const notifier = path.join(repoRoot, 'scripts/notify-operator-docs.sh');
const notifyWorkflow = path.join(repoRoot, '.github/workflows/notify-operator-docs.yml');
const releaseWorkflow = path.join(repoRoot, '.github/workflows/release.yml');
const chartWorkflow = path.join(repoRoot, '.github/workflows/release-chart.yml');

function createOperatorTagFixture(t, chartWorkflowContent, tags) {
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-tags-'));
  t.after(() => fs.rmSync(repo, { recursive: true, force: true }));
  const workflowPath = path.join(repo, '.github/workflows/release-chart.yml');
  fs.mkdirSync(path.dirname(workflowPath), { recursive: true });
  fs.writeFileSync(workflowPath, chartWorkflowContent);
  execFileSync('git', ['init', '-q', repo]);
  execFileSync('git', ['-C', repo, 'config', 'user.name', 'Operator docs test']);
  execFileSync('git', ['-C', repo, 'config', 'user.email', 'operator-docs-test@example.invalid']);
  execFileSync('git', ['-C', repo, 'add', '.']);
  execFileSync('git', ['-C', repo, 'commit', '-q', '-m', 'fixture']);
  const headSha = execFileSync('git', ['-C', repo, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
  for (const tag of tags) execFileSync('git', ['-C', repo, 'tag', tag]);
  return { repo, workflowPath, headSha };
}

function createFakeGh(temp) {
  const fakeBin = path.join(temp, 'bin');
  fs.mkdirSync(fakeBin);
  const capture = path.join(temp, 'gh-args.txt');
  const fakeGh = path.join(fakeBin, 'gh');
  fs.writeFileSync(fakeGh, '#!/usr/bin/env bash\nprintf \'%s\\n\' "$@" > "$GH_ARGS_CAPTURE"\n');
  fs.chmodSync(fakeGh, 0o755);
  return { capture, path: `${fakeBin}${path.delimiter}${process.env.PATH}` };
}

test('dispatches the workflow input or explicit tag through the agreed event contract', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-notifier-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const fakeBin = path.join(temp, 'bin');
  fs.mkdirSync(fakeBin);
  const capture = path.join(temp, 'gh-args.txt');
  const fakeGh = path.join(fakeBin, 'gh');
  fs.writeFileSync(fakeGh, '#!/usr/bin/env bash\nprintf \'%s\\n\' "$@" > "$GH_ARGS_CAPTURE"\n');
  fs.chmodSync(fakeGh, 0o755);

  const cases = [
    {
      env: { INPUT_OPERATOR_TAG: 'v1.2.3', PUSHED_TAG: 'v8.8.8' },
      expectedTag: 'v1.2.3',
    },
    {
      env: { INPUT_OPERATOR_TAG: '', PUSHED_TAG: 'v1.2.5' },
      expectedTag: 'v1.2.5',
    },
  ];

  for (const { env, expectedTag } of cases) {
    execFileSync('bash', [notifier], {
      env: {
        ...process.env,
        ...env,
        GH_ARGS_CAPTURE: capture,
        PATH: `${fakeBin}${path.delimiter}${process.env.PATH}`,
      },
    });

    assert.deepEqual(fs.readFileSync(capture, 'utf8').trim().split('\n'), [
      'api',
      '--method',
      'POST',
      'repos/krkn-chaos/website/dispatches',
      '-f',
      'event_type=operator-docs-release-ready',
      '-F',
      `client_payload[operator_tag]=${expectedTag}`,
    ]);
  }
});

test('fails before dispatch when the release tag is missing', () => {
  const result = spawnSync('bash', [notifier], {
    env: {
      ...process.env,
      INPUT_OPERATOR_TAG: '',
      PUSHED_TAG: '',
    },
    encoding: 'utf8',
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /OPERATOR_TAG must be set/);
});

test('skips an RC tag based on its tag even if release metadata is unavailable', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-notifier-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const fakeBin = path.join(temp, 'bin');
  fs.mkdirSync(fakeBin);
  const capture = path.join(temp, 'gh-args.txt');
  const fakeGh = path.join(fakeBin, 'gh');
  fs.writeFileSync(fakeGh, '#!/usr/bin/env bash\nprintf \'%s\\n\' "$@" > "$GH_ARGS_CAPTURE"\n');
  fs.chmodSync(fakeGh, 0o755);

  const result = spawnSync('bash', [notifier], {
    env: {
      ...process.env,
      INPUT_OPERATOR_TAG: 'v1.1.0-rc7',
      GH_ARGS_CAPTURE: capture,
      PATH: `${fakeBin}${path.delimiter}${process.env.PATH}`,
    },
    encoding: 'utf8',
  });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /Skipping non-stable Operator release tag/i);
  assert.equal(fs.existsSync(capture), false);
});

test('successful chart publication calls the reusable notifier with the exact tag', () => {
  const release = fs.readFileSync(releaseWorkflow, 'utf8');
  const chart = fs.readFileSync(chartWorkflow, 'utf8');
  const notifierConfig = fs.readFileSync(notifyWorkflow, 'utf8');

  assert.doesNotMatch(release, /^  notify-operator-docs:/m);
  assert.match(chart, /^permissions:\n  contents: read$/m);
  assert.match(chart, /notify-website:\n\s+name: Notify Website\n\s+needs: release-chart\n\s+if: \$\{\{ needs\.release-chart\.result == 'success' \}\}\n\s+uses: \.\/\.github\/workflows\/notify-operator-docs\.yml\n\s+with:\n\s+operator_tag: \$\{\{ github\.ref_name \}\}\n\s+secrets:\n\s+DOC_SYNC_BOT_APP_PRIVATE_KEY: \$\{\{ secrets\.DOC_SYNC_BOT_APP_PRIVATE_KEY \}\}/m);
  assert.match(notifierConfig, /workflow_run:/);
  assert.match(notifierConfig, /workflows: \["Release Helm Chart"\]/);
  assert.doesNotMatch(notifierConfig, /workflows: \["Release Helm Chart", "Release"\]/);
  assert.match(notifierConfig, /github\.event\.workflow_run\.event == 'push'/);
  assert.match(notifierConfig, /github\.event\.workflow_run\.conclusion == 'success'/);
  assert.match(notifierConfig, /github\.event\.workflow_run\.head_repository\.full_name/);
  assert.match(notifierConfig, /ref: main/);
  assert.match(notifierConfig, /fetch-depth: 0/);
  assert.match(notifierConfig, /INPUT_OPERATOR_TAG: \$\{\{ inputs\.operator_tag \}\}/);
  assert.doesNotMatch(notifierConfig, /release:\n\s+types: \[published\]/);
  assert.match(notifierConfig, /PUSHED_TAG: \$\{\{ github\.event\.workflow_run\.head_branch \}\}/);
  assert.match(notifierConfig, /WORKFLOW_RUN_HEAD_SHA: \$\{\{ github\.event\.workflow_run\.head_sha \}\}/);
  assert.match(notifierConfig, /SOURCE_RELEASE_CHART: operator-tag-source\/\.github\/workflows\/release-chart\.yml/);
  assert.ok(notifierConfig.indexOf('Detect whether the tag workflow already notified directly') < notifierConfig.indexOf('Mint website dispatch token'));
});

test('default-branch fallback does not dispatch for a source workflow with its direct notifier job', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-fallback-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const chart = `jobs:\n  release-chart:\n    runs-on: ubuntu-latest\n  notify-website:\n    needs: release-chart\n    uses: ./.github/workflows/notify-operator-docs.yml\n`;
  const fixture = createOperatorTagFixture(t, chart, ['v1.2.3']);
  const fakeGh = createFakeGh(temp);
  const result = spawnSync('bash', [notifier], {
    cwd: repoRoot,
    env: {
      ...process.env,
      GITHUB_EVENT_NAME: 'workflow_run',
      INPUT_OPERATOR_TAG: '',
      PUSHED_TAG: 'v1.2.3',
      WORKFLOW_RUN_HEAD_SHA: fixture.headSha,
      OPERATOR_REPO_ROOT: fixture.repo,
      SOURCE_RELEASE_CHART: fixture.workflowPath,
      GH_ARGS_CAPTURE: fakeGh.capture,
      PATH: fakeGh.path,
    },
    encoding: 'utf8',
  });

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /already called the notifier directly; skipping fallback dispatch/i);
  assert.equal(fs.existsSync(fakeGh.capture), false);
});

test('legacy workflow-run fallback resolves a tag by head SHA instead of treating main as the release tag', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-fallback-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const fixture = createOperatorTagFixture(t, 'jobs:\n  release-chart:\n    runs-on: ubuntu-latest\n', ['v1.2.3']);
  const fakeGh = createFakeGh(temp);
  const result = spawnSync('bash', [notifier], {
    cwd: repoRoot,
    env: {
      ...process.env,
      GITHUB_EVENT_NAME: 'workflow_run',
      INPUT_OPERATOR_TAG: '',
      PUSHED_TAG: 'main',
      WORKFLOW_RUN_HEAD_SHA: fixture.headSha,
      OPERATOR_REPO_ROOT: fixture.repo,
      SOURCE_RELEASE_CHART: fixture.workflowPath,
      GH_ARGS_CAPTURE: fakeGh.capture,
      PATH: fakeGh.path,
    },
    encoding: 'utf8',
  });

  assert.equal(result.status, 0, result.stderr);
  assert.equal(fs.existsSync(fakeGh.capture), true, result.stderr);
  assert.deepEqual(fs.readFileSync(fakeGh.capture, 'utf8').trim().split('\n'), [
    'api',
    '--method',
    'POST',
    'repos/krkn-chaos/website/dispatches',
    '-f',
    'event_type=operator-docs-release-ready',
    '-F',
    'client_payload[operator_tag]=v1.2.3',
  ]);
});

test('legacy fallback honors the original tag when several tags point to the workflow run head', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-fallback-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const fixture = createOperatorTagFixture(t, 'jobs:\n  release-chart:\n    runs-on: ubuntu-latest\n', ['v1.2.3', 'v1.2.4']);
  const fakeGh = createFakeGh(temp);
  const result = spawnSync('bash', [notifier], {
    cwd: repoRoot,
    env: {
      ...process.env,
      GITHUB_EVENT_NAME: 'workflow_run',
      INPUT_OPERATOR_TAG: '',
      PUSHED_TAG: 'v1.2.4',
      WORKFLOW_RUN_HEAD_SHA: fixture.headSha,
      OPERATOR_REPO_ROOT: fixture.repo,
      SOURCE_RELEASE_CHART: fixture.workflowPath,
      GH_ARGS_CAPTURE: fakeGh.capture,
      PATH: fakeGh.path,
    },
    encoding: 'utf8',
  });

  assert.equal(result.status, 0, result.stderr);
  assert.match(fs.readFileSync(fakeGh.capture, 'utf8'), /client_payload\[operator_tag\]=v1\.2\.4/);
});

test('legacy fallback refuses to dispatch when the workflow head has no tag', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-fallback-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const fixture = createOperatorTagFixture(t, 'jobs:\n  release-chart:\n    runs-on: ubuntu-latest\n', []);
  const fakeGh = createFakeGh(temp);
  const result = spawnSync('bash', [notifier], {
    cwd: repoRoot,
    env: {
      ...process.env,
      GITHUB_EVENT_NAME: 'workflow_run',
      INPUT_OPERATOR_TAG: '',
      PUSHED_TAG: 'main',
      WORKFLOW_RUN_HEAD_SHA: fixture.headSha,
      OPERATOR_REPO_ROOT: fixture.repo,
      SOURCE_RELEASE_CHART: fixture.workflowPath,
      GH_ARGS_CAPTURE: fakeGh.capture,
      PATH: fakeGh.path,
    },
    encoding: 'utf8',
  });

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /no tag points to workflow run head SHA/i);
  assert.equal(fs.existsSync(fakeGh.capture), false);
});

test('legacy fallback fails closed when several tags point to the run SHA and head branch is unusable', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-fallback-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const fixture = createOperatorTagFixture(t, 'jobs:\n  release-chart:\n    runs-on: ubuntu-latest\n', ['v1.2.3', 'v1.2.4']);
  const fakeGh = createFakeGh(temp);
  const result = spawnSync('bash', [notifier], {
    cwd: repoRoot,
    env: {
      ...process.env,
      GITHUB_EVENT_NAME: 'workflow_run',
      INPUT_OPERATOR_TAG: '',
      PUSHED_TAG: 'main',
      WORKFLOW_RUN_HEAD_SHA: fixture.headSha,
      OPERATOR_REPO_ROOT: fixture.repo,
      SOURCE_RELEASE_CHART: fixture.workflowPath,
      GH_ARGS_CAPTURE: fakeGh.capture,
      PATH: fakeGh.path,
    },
    encoding: 'utf8',
  });

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /multiple tags point to workflow run head SHA/i);
  assert.equal(fs.existsSync(fakeGh.capture), false);
});

test('the metadata template uses the same tag-keyed releases format as the Website receiver', () => {
  const metadata = fs.readFileSync(path.join(repoRoot, 'docs/website-release.yaml'), 'utf8');
  assert.match(metadata, /^schema_version: 1$/m);
  assert.match(metadata, /^releases: \[\]$/m);
  assert.doesNotMatch(metadata, /^website_commit:/m);
});

test('documents the GitHub App settings and per-tag release metadata', () => {
  const readme = fs.readFileSync(path.join(repoRoot, 'README.md'), 'utf8');
  assert.match(readme, /DOC_SYNC_BOT_APP_ID/);
  assert.match(readme, /DOC_SYNC_BOT_APP_PRIVATE_KEY/);
  assert.match(readme, /docs\/website-release\.yaml/);
});
