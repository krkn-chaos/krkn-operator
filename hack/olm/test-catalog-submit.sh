#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
test_dir=$(mktemp -d "${TMPDIR:-/tmp}/catalog-submit-test.XXXXXX")
trap 'rm -rf "$test_dir"' EXIT

git init --bare --quiet "$test_dir/origin.git"
git init --quiet -b main "$test_dir/catalog"
git -C "$test_dir/catalog" config user.name test
git -C "$test_dir/catalog" config user.email test@example.com
git -C "$test_dir/catalog" config commit.gpgSign false
printf '%s\n' initial > "$test_dir/catalog/README.md"
git -C "$test_dir/catalog" add README.md
git -C "$test_dir/catalog" commit --quiet -m initial
git -C "$test_dir/catalog" remote add origin "$test_dir/origin.git"
git -C "$test_dir/catalog" push --quiet -u origin main

mkdir -p "$test_dir/bin"
cat > "$test_dir/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$1 $2" == "pr list" ]]; then
  exit 0
fi
if [[ "$1 $2" == "pr create" ]]; then
  touch "$GH_TEST_PR_CREATED"
  exit 0
fi
echo "unexpected gh invocation: $*" >&2
exit 1
EOF
chmod +x "$test_dir/bin/gh"

# shellcheck disable=SC1091
source "$script_dir/catalog-submit.sh"
export PATH="$test_dir/bin:$PATH"
export GH_TEST_PR_CREATED="$test_dir/pr-created"
export GH_TOKEN=test-token
CATALOG_DIR="$test_dir/catalog"
CATALOG_REPOSITORY=test/catalog
CATALOG_FORK_OWNER=test

printf '%s\n' generated > "$test_dir/catalog/generated.yaml"
catalog_submit_open_pr \
  "operator: test catalog submission" \
  "test catalog submission" \
  "$test_dir/pr-body.md" \
  generated.yaml

test -e "$GH_TEST_PR_CREATED"
test "$(git --git-dir "$test_dir/origin.git" show main:generated.yaml)" = generated
echo "Catalog submission helper tests passed"
