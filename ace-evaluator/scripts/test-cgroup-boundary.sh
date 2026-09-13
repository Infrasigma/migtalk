#!/usr/bin/env bash
set -euo pipefail

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT
mkdir -p "$ROOT/ace.slice/ace-evaluator.slice/runner.service"

write_limits() {
  local path="$1" cpu="$2" memory="$3" swap="$4" pids="$5"
  printf '%s\n' "$cpu" >"$path/cpu.max"
  printf '%s\n' "$memory" >"$path/memory.max"
  printf '%s\n' "$swap" >"$path/memory.swap.max"
  printf '%s\n' "$pids" >"$path/pids.max"
}

slice="$ROOT/ace.slice/ace-evaluator.slice"
runner="$slice/runner.service"
child="$runner/ace-scientific-workload-test"

write_limits "$slice" '100000 100000' '1073741824' '0' '256'
write_limits "$runner" 'max 100000' 'max' 'max' '1649'

ACE_CGROUP_ROOT="$ROOT" source "$(dirname "$0")/check-cgroup-boundary.sh"

# Valid: finite enforcing ancestor + max runner leaf is intentionally accepted.
check_runner_boundary '/ace.slice/ace-evaluator.slice/runner.service'

# Valid workload boundary: explicit child limits plus finite ancestor.
mkdir -p "$child"
write_limits "$child" '100000 100000' '1073741824' '0' '256'
check_workload_boundary '/ace.slice/ace-evaluator.slice/runner.service' '/ace.slice/ace-evaluator.slice/runner.service/ace-scientific-workload-test'

# Invalid: ancestor without finite CPU limit must fail despite the runner leaf being finite.
printf '%s\n' 'max 100000' >"$slice/cpu.max"
if check_runner_boundary '/ace.slice/ace-evaluator.slice/runner.service'; then
  echo 'expected unconstrained ancestor rejection' >&2
  exit 1
fi
printf '%s\n' '100000 100000' >"$slice/cpu.max"

# Invalid: wrong slice path.
if check_runner_boundary '/ace.slice/other.slice/runner.service'; then
  echo 'expected wrong-slice rejection' >&2
  exit 1
fi

# Invalid: wrong cgroup path component must not be accepted as ACE slice.
if check_runner_boundary '/ace.slice/ace-evaluator.slice-other/runner.service'; then
  echo 'expected wrong-path rejection' >&2
  exit 1
fi

# Invalid: swap enabled at enforcing ancestor must fail.
printf '%s\n' '4096' >"$slice/memory.swap.max"
if check_runner_boundary '/ace.slice/ace-evaluator.slice/runner.service'; then
  echo 'expected ancestor swap rejection' >&2
  exit 1
fi

# Restore and verify the workload-specific boundary rejects an incorrect child limit.
printf '%s\n' '0' >"$slice/memory.swap.max"
printf '%s\n' 'max' >"$child/memory.max"
if check_workload_boundary '/ace.slice/ace-evaluator.slice/runner.service' '/ace.slice/ace-evaluator.slice/runner.service/ace-scientific-workload-test'; then
  echo 'expected workload limit rejection' >&2
  exit 1
fi

echo 'cgroup hierarchical boundary regression tests: PASS'