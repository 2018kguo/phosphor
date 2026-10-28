#!/usr/bin/env bash
set -euo pipefail

if [[ $# != 3 ]]; then
  echo 'usage: ghostty/build.sh BASELINE_REF CANDIDATE_REF OUTPUT_DIR' >&2
  exit 2
fi
: "${GHOSTTY_SOURCE:?Set GHOSTTY_SOURCE to the native source used to build the library}"
: "${CGO_CFLAGS:?Set CGO_CFLAGS to the native include directory}"
: "${CGO_LDFLAGS:?Set CGO_LDFLAGS to the native library and runtime search path}"
root=$(cd "$(dirname "$0")/../.." && pwd)
baseline=$(git -C "$root" rev-parse --verify "$1^{commit}")
candidate=$(git -C "$root" rev-parse --verify "$2^{commit}")
mkdir -p "$3"
output=$(cd "$3" && pwd)
temporary=$(mktemp -d)
cleanup() {
  for label in baseline candidate; do
    git -C "$root" worktree remove --force "$temporary/$label" >/dev/null 2>&1 || true
  done
  rm -rf "$temporary"
}
trap cleanup EXIT
export GOWORK=off
cd "$root/oracle"

for label in baseline candidate; do
  revision=$baseline
  if [[ $label == candidate ]]; then revision=$candidate; fi
  core=$temporary/$label
  git -C "$root" worktree add --quiet --detach "$core" "$revision"
  # This package belongs to the measurement input, not the production core.
  # Refuse unequal fixtures rather than silently changing the work per build.
  cmp "$root/internal/benchdata/benchdata.go" "$core/internal/benchdata/benchdata.go"
  mkdir "$temporary/$label-module"
  cp "$root/oracle/go.mod" "$root/oracle/go.sum" "$temporary/$label-module/"
  modfile=$temporary/$label-module/go.mod
  go mod edit -modfile="$modfile" -replace="github.com/2018kguo/phosphor=$core"
  (
    cd "$root/oracle"
    go build -mod=mod -modfile="$modfile" -tags ghostty -o "$output/$label" ./ghostty
  )
  go version -m "$output/$label" > "$output/$label-build.txt"
  git -C "$root" ls-tree -r "$revision" > "$output/$label-source.txt"
done

{
  echo "baseline=$baseline"
  echo "candidate=$candidate"
  echo "driver=$(git -C "$root" rev-parse HEAD)"
  echo "native=$(git -C "$GHOSTTY_SOURCE" rev-parse HEAD)"
  go version
  for source in "$root"/oracle/ghostty/*.{go,c,h,json,sh}; do
    [[ -f $source ]] || continue
    echo "$(git -C "$root" hash-object "$source") ${source#"$root/"}"
  done
  git -C "$root" status --porcelain -- oracle internal/benchdata
  git -C "$GHOSTTY_SOURCE" status --porcelain --untracked-files=no
} > "$output/sources.txt"
echo "Built matched drivers in $output; inspect sources.txt for dirty source before measuring."
