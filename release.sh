#!/usr/bin/env bash

set -euo pipefail

readonly REPOSITORY="socure-inc/riskos-cli"
readonly GH_HOSTNAME="github.com"

usage() {
  echo "Usage: $0 <version>"
  echo "Example: $0 v1.0.7"
}

fail() {
  echo "Error: $*" >&2
  exit 1
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

readonly VERSION="$1"
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  fail "version must use the format vMAJOR.MINOR.PATCH"
fi

for command in git go gh tar awk; do
  command -v "$command" >/dev/null 2>&1 || fail "required command not found: $command"
done

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" ||
  fail "run this script from inside the riskos-cli repository"
readonly ROOT
cd "$ROOT"

export GH_HOST="$GH_HOSTNAME"

if [[ -n "$(git status --porcelain --untracked-files=normal)" ]]; then
  fail "working tree is dirty; commit, stash, or remove local changes before releasing"
fi

echo "Fetching tags from origin..."
git fetch origin --tags

if git rev-parse --verify --quiet "refs/tags/$VERSION" >/dev/null; then
  echo "Tag $VERSION exists locally."
else
  echo "Creating annotated tag $VERSION at HEAD..."
  git tag -a "$VERSION" -m "Release $VERSION"
fi

REMOTE_TAG="$(git ls-remote --tags origin "refs/tags/$VERSION" | awk '{print $1}')"
readonly REMOTE_TAG
if [[ -z "$REMOTE_TAG" ]]; then
  echo "Pushing tag $VERSION to origin..."
  git push origin "$VERSION"
else
  LOCAL_TAG="$(git rev-parse "refs/tags/$VERSION")"
  readonly LOCAL_TAG
  if [[ "$LOCAL_TAG" != "$REMOTE_TAG" ]]; then
    fail "local and origin tags for $VERSION point to different objects"
  fi
  echo "Tag $VERSION already exists on origin."
fi

gh auth status --hostname "$GH_HOSTNAME" >/dev/null ||
  fail "authenticate first with: gh auth login --hostname $GH_HOSTNAME --git-protocol ssh"

if gh release view "$VERSION" --repo "$REPOSITORY" >/dev/null 2>&1; then
  fail "release $VERSION already exists"
fi
echo "Release $VERSION does not exist yet."

readonly DIST="$ROOT/dist/$VERSION"
readonly LDFLAGS="-s -w -X main.version=$VERSION"

BUILD_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/riskos-release.XXXXXX")"
readonly BUILD_ROOT
trap 'rm -rf "$BUILD_ROOT"' EXIT

echo "Extracting tagged source..."
git archive "$VERSION" | tar -x -C "$BUILD_ROOT"

echo "Testing and building $VERSION..."
mkdir -p "$DIST"
(
  cd "$BUILD_ROOT"
  go test ./...
  GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "$DIST/riskos-darwin-arm64" .
  GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$DIST/riskos-darwin-amd64" .
  GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$DIST/riskos-linux-amd64" .
  GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "$DIST/riskos-linux-arm64" .
  GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$DIST/riskos-windows-amd64.exe" .
)

readonly ASSETS=(
  "$DIST/riskos-darwin-arm64"
  "$DIST/riskos-darwin-amd64"
  "$DIST/riskos-linux-amd64"
  "$DIST/riskos-linux-arm64"
  "$DIST/riskos-windows-amd64.exe"
)

echo "Creating GitHub release $VERSION..."
gh release create "$VERSION" \
  --repo "$REPOSITORY" \
  --verify-tag \
  --generate-notes \
  --title "$VERSION" \
  "${ASSETS[@]}"

echo "Release $VERSION created successfully."
