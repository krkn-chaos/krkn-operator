'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execFileSync, spawnSync } = require('node:child_process');
const test = require('node:test');

const repoRoot = path.resolve(__dirname, '../..');
const notifier = path.join(repoRoot, 'scripts/notify-operator-docs.sh');

test('dispatches the published tag to the website receiver with the agreed event contract', (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'operator-docs-notifier-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const fakeBin = path.join(temp, 'bin');
  fs.mkdirSync(fakeBin);
  const capture = path.join(temp, 'gh-args.txt');
  const fakeGh = path.join(fakeBin, 'gh');
  fs.writeFileSync(fakeGh, '#!/usr/bin/env bash\nprintf \'%s\\n\' "$@" > "$GH_ARGS_CAPTURE"\n');
  fs.chmodSync(fakeGh, 0o755);

  execFileSync('bash', [notifier], {
    env: {
      ...process.env,
      OPERATOR_TAG: 'v1.2.3-rc1',
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
    'client_payload[operator_tag]=v1.2.3-rc1',
  ]);
});

test('fails before dispatch when the release tag is missing', () => {
  const result = spawnSync('bash', [notifier], {
    env: { ...process.env, OPERATOR_TAG: '' },
    encoding: 'utf8',
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /OPERATOR_TAG must be set/);
});
