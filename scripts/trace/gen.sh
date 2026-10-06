#!/usr/bin/env bash
# Regenerate the golden traces of one example from the JS reference:
#   scripts/trace/gen.sh <upstream-example>
# examples/manifest.tsv maps the upstream name to its Go package directory.
# Each scripts/trace/<example>/<name>.ts prints JSON on stdout and is
# written to that package's testdata/<name>.golden.json. A script named
# <name>.stdout.ts is recorded as plain text to testdata/<name>.stdout.txt.
set -euo pipefail
cd "$(dirname "$0")"
ex="${1:?usage: gen.sh <upstream-example>}"
package=$(awk -F '\t' -v example="$ex" '$1 == example { print $2 }' ../../examples/manifest.tsv)
if [[ -z "$package" || ! -d "$ex" ]]; then
  echo "No trace scripts for example: $ex" >&2
  exit 1
fi
out="../../examples/$package/testdata"
mkdir -p "$out"
shopt -s nullglob
scripts=("$ex"/*.ts)
if (( ${#scripts[@]} == 0 )); then
  echo "No trace scripts for example: $ex" >&2
  exit 1
fi
tmp=""
trap '[[ -z "$tmp" ]] || rm -f "$tmp"' EXIT
for f in "${scripts[@]}"; do
  n=$(basename "$f" .ts)
  case "$n" in
    *.stdout) target="$out/${n%.stdout}.stdout.txt" ;;
    *)        target="$out/$n.golden.json" ;;
  esac
  # A failed recorder must leave the existing fixture intact.
  tmp=$(mktemp "$out/.trace.XXXXXX")
  bun run "$f" > "$tmp"
  mv "$tmp" "$target"
  tmp=""
done
