#!/usr/bin/env bash
# Run every Test function of a package in its own process (a panic or
# crash only fails that test) and print PASS/FAIL/SKIP counts.
#
#   scripts/status.sh [pkg-dir] [name-regex]
#   e.g. scripts/status.sh ./xstate '^TestInvoke_'
#
# Per-run lists: /tmp/xs-status/run.*/<pkg>.{pass,fail,skip} (path printed at the end);
# the latest copy is also at /tmp/xs-status/<pkg>.{pass,fail,skip}. Set RACE=1 to build with -race, JOBS=n for parallelism.
set -u
pkg="${1:-./xstate}"; re="${2:-.}"
cd "$(dirname "$0")/.."
base=/tmp/xs-status; mkdir -p "$base"
out=$(mktemp -d "$base/run.XXXXXX")   # per-invocation dir: concurrent runs never share files
name=$(echo "$pkg" | tr '/.' '__'); name=${name:-root}
bin="$out/$name.test"
flags=""; [ "${RACE:-0}" = 1 ] && flags="-race"
go test $flags -c -o "$bin" "$pkg" 2>&1 | head -40 || exit 1
[ -x "$bin" ] || { echo "BUILD FAILED"; exit 2; }
tests=$(cd "$pkg" && "$bin" -test.list '.*' | grep '^Test' | grep -E "$re")
: > "$out/$name.fail"; : > "$out/$name.skip"; : > "$out/$name.pass"
run_one() {
  t="$1"; bin="$2"; pkg="$3"; out="$4"; name="$5"
  res=$(cd "$pkg" && timeout 30 "$bin" -test.run "^$t\$" -test.v 2>&1)
  if echo "$res" | grep -q -- "--- SKIP: $t"; then echo "$t" >> "$out/$name.skip"
  elif echo "$res" | grep -q -- "--- PASS: $t"; then echo "$t" >> "$out/$name.pass"
  else echo "$t" >> "$out/$name.fail"; fi
}
export -f run_one
echo "$tests" | xargs -P "${JOBS:-8}" -I{} bash -c 'run_one {} '"$bin $pkg $out $name"
p=$(wc -l < "$out/$name.pass"); f=$(wc -l < "$out/$name.fail"); s=$(wc -l < "$out/$name.skip")
echo "pkg=$pkg PASS=$p FAIL=$f SKIP=$s TOTAL=$((p+f+s))"
# keep the newest lists at a stable path
for k in pass fail skip; do cp "$out/$name.$k" "$base/$name.$k"; done
echo "lists: $out/$name.{pass,fail,skip}"
