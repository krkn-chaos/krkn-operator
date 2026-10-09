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

test('dispatches the workflow input, release tag, or pushed tag through the agreed event contract', (t) => {
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
      env: { INPUT_OPERATOR_TAG: 'v1.2.3', RELEASE_TAG: 'v9.9.9', PUSHED_TAG: 'v8.8.8' },
      expectedTag: 'v1.2.3',
    },
    {
      env: { INPUT_OPERATOR_TAG: '', RELEASE_TAG: 'v1.2.4', PUSHED_TAG: 'v8.8.8' },
      expectedTag: 'v1.2.4',
    },
    {
      env: { INPUT_OPERATOR_TAG: '', RELEASE_TAG: '', PUSHED_TAG: 'v1.2.5' },
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
      RELEASE_TAG: '',
      PUSHED_TAG: '',
    },
    encoding: 'utf8',
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /OPERATOR_TAG must be set/);
});

test('skips an RC tag even when GitHub incorrectly marks it as stable', (t) => {
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
      RELEASE_IS_PRERELEASE: 'false',
      GH_ARGS_CAPTURE: capture,
      PATH: `${fakeBin}${path.delimiter}${process.env.PATH}`,
    },
    encoding: 'utf8',
  });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /Skipping non-stable Operator release tag/i);
  assert.equal(fs.existsSync(capture), false);
});

test('skips a published release when GitHub marks it as a prerelease', (t) => {
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
      RELEASE_TAG: 'v1.2.3',
      RELEASE_IS_PRERELEASE: 'true',
      GH_ARGS_CAPTURE: capture,
      PATH: `${fakeBin}${path.delimiter}${process.env.PATH}`,
    },
    encoding: 'utf8',
  });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /GitHub marks .* as a prerelease/i);
  assert.equal(fs.existsSync(capture), false);
});

test('trusted default-branch notifications follow publication completion across release branches', () => {
  const release = fs.readFileSync(releaseWorkflow, 'utf8');
  const chart = fs.readFileSync(chartWorkflow, 'utf8');
  const notifierConfig = fs.readFileSync(notifyWorkflow, 'utf8');

  assert.doesNotMatch(release, /^  notify-operator-docs:/m);
  assert.doesNotMatch(chart, /^  notify-operator-docs:/m);
  assert.match(notifierConfig, /workflow_run:/);
  assert.match(notifierConfig, /workflows: \["Release Helm Chart", "Release"\]/);
  assert.match(notifierConfig, /github\.event\.workflow_run\.head_repository\.full_name/);
  assert.match(notifierConfig, /ref: main/);
  assert.match(notifierConfig, /INPUT_OPERATOR_TAG: \$\{\{ inputs\.operator_tag \}\}/);
  assert.match(notifierConfig, /RELEASE_TAG: \$\{\{ github\.event\.release\.tag_name \}\}/);
  assert.match(notifierConfig, /PUSHED_TAG: \$\{\{ github\.event\.workflow_run\.head_branch \|\| github\.ref_name \}\}/);
  assert.match(notifierConfig, /RELEASE_IS_PRERELEASE: \$\{\{ github\.event\.release\.prerelease \}\}/);
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
