#!/usr/bin/env bash
set -euo pipefail
PYTHON=${PYTHON:-python3}
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
git config user.name 'github-actions[bot]'
git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
git fetch origin main
test "$(git rev-parse origin/main)" = "$GITHUB_SHA" # release the reviewed tip, not an outdated run
if git rev-parse "refs/tags/v$VERSION" >/dev/null 2>&1; then
  # Resume a release whose Git push succeeded but GitHub Release creation failed.
  git merge-base --is-ancestor "$GITHUB_SHA" "v$VERSION"
  test "$(git rev-parse "v$VERSION^{tree}")" = "$(git rev-parse "$GITHUB_SHA^{tree}")"
else
  git checkout -b release-candidate "$GITHUB_SHA"
  if git ls-remote --exit-code --heads origin stable >/dev/null; then
    git fetch origin stable
    old=$(git show origin/stable:fve_controller/config.yaml | sed -n 's/^version: "\(.*\)"$/\1/p')
    "$PYTHON" -c 'import sys; assert tuple(map(int,sys.argv[2].split("."))) > tuple(map(int,sys.argv[1].split("."))), "Version must increase"' "$old" "$VERSION"
    # Keep stable history while promoting exactly the tested main tree.
    git merge --no-commit --no-ff -s ours origin/stable
    git commit --allow-empty -m "Release $VERSION from $GITHUB_SHA"
  fi
  test "$(git rev-parse HEAD^{tree})" = "$(git rev-parse "$GITHUB_SHA^{tree}")"
  git tag "v$VERSION"
  git push --atomic origin HEAD:refs/heads/stable "refs/tags/v$VERSION"
fi
if ! gh release view "v$VERSION" >/dev/null 2>&1; then
  gh release create "v$VERSION" --verify-tag --title "FVE Controller $VERSION" --notes-file fve_controller/CHANGELOG.md
fi
