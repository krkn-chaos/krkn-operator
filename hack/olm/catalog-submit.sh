#!/usr/bin/env bash

# Shared GitHub catalog fork and pull-request plumbing. Catalog-specific
# scripts prepare the package files, then call these functions to synchronize
# the fork and create or update the upstream pull request.

catalog_submit_checkout() {
  local fork=$1
  local repository=$2
  local branch=$3

  [[ "$fork" =~ ^[^/]+/[^/]+$ ]] || {
    echo "catalog fork must have the form <owner>/<repository>" >&2
    return 2
  }
  : "${GH_TOKEN:?GH_TOKEN must be configured with permission to push to the catalog fork and open upstream PRs}"

  git config --global user.name "github-actions[bot]"
  git config --global user.email "41898282+github-actions[bot]@users.noreply.github.com"
  gh auth setup-git

  CATALOG_FORK="$fork"
  CATALOG_REPOSITORY="$repository"
  CATALOG_FORK_OWNER=${fork%%/*}
  CATALOG_WORK_DIR=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/community-operators.XXXXXX")
  CATALOG_DIR="$CATALOG_WORK_DIR/catalog"

  gh repo clone "$CATALOG_FORK" "$CATALOG_DIR" >/dev/null
  if git -C "$CATALOG_DIR" remote get-url upstream >/dev/null 2>&1; then
    git -C "$CATALOG_DIR" remote set-url upstream "https://github.com/$CATALOG_REPOSITORY.git"
  else
    git -C "$CATALOG_DIR" remote add upstream "https://github.com/$CATALOG_REPOSITORY.git"
  fi
  git -C "$CATALOG_DIR" remote set-url origin "https://github.com/$CATALOG_FORK.git"
  git -C "$CATALOG_DIR" fetch --quiet upstream main
  git -C "$CATALOG_DIR" checkout --quiet -B "$branch" upstream/main
}

catalog_submit_open_pr() {
  local commit_message=$1
  local title=$2
  local body_file=$3
  shift 3
  local branch
  branch=$(git -C "$CATALOG_DIR" branch --show-current)

  git -C "$CATALOG_DIR" add "$@"
  git -C "$CATALOG_DIR" commit -m "$commit_message"
  git -C "$CATALOG_DIR" push --force-with-lease origin "$branch"

  existing_pr=$(gh pr list \
    --repo "$CATALOG_REPOSITORY" \
    --head "$CATALOG_FORK_OWNER:$branch" \
    --state open \
    --json number \
    --jq '.[0].number // empty')
  if [[ -n "$existing_pr" ]]; then
    echo "Updated existing catalog PR #$existing_pr"
    return 0
  fi

  gh pr create \
    --repo "$CATALOG_REPOSITORY" \
    --head "$CATALOG_FORK_OWNER:$branch" \
    --base main \
    --title "$title" \
    --body-file "$body_file"
}
