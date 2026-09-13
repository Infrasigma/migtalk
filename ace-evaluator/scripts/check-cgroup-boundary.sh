#!/usr/bin/env bash
set -euo pipefail

ACE_SLICE="ace-evaluator.slice"
ACE_CPU_MAX="100000 100000"
ACE_MEMORY_MAX="1073741824"
ACE_SWAP_MAX="0"
ACE_PIDS_MAX="256"
ACE_CGROUP_ROOT="${ACE_CGROUP_ROOT:-/sys/fs/cgroup}"

fail() {
  echo "ACE-CGROUP-BOUNDARY: FAIL: $*" >&2
  return 1
}

finite_first_field() {
  local value="$1"
  [[ "${value%% *}" != "max" ]]
}

check_slice_limits() {
  local root="$1"
  [[ -d "$root" ]] || { fail "cgroup directory does not exist: $root"; return 1; }
  [[ -r "$root/cpu.max" ]] || { fail "missing cpu.max at $root"; return 1; }
  [[ -r "$root/memory.max" ]] || { fail "missing memory.max at $root"; return 1; }
  [[ -r "$root/memory.swap.max" ]] || { fail "missing memory.swap.max at $root"; return 1; }
  [[ -r "$root/pids.max" ]] || { fail "missing pids.max at $root"; return 1; }

  local cpu memory swap pids
  cpu="$(cat "$root/cpu.max")"
  memory="$(cat "$root/memory.max")"
  swap="$(cat "$root/memory.swap.max")"
  pids="$(cat "$root/pids.max")"

  finite_first_field "$cpu" || { fail "CPU is unlimited at enforcing cgroup: $root"; return 1; }
  finite_first_field "$memory" || { fail "memory is unlimited at enforcing cgroup: $root"; return 1; }
  finite_first_field "$pids" || { fail "pids are unlimited at enforcing cgroup: $root"; return 1; }
  [[ "$swap" == 0 ]] || { fail "swap is not disabled at enforcing cgroup $root: $swap"; return 1; }
}

runner_slice_path() {
  local runner_path="$1"
  [[ "$runner_path" == /* && "$runner_path" != *..* ]] || { fail "invalid absolute runner cgroup path: $runner_path"; return 1; }
  local slice_path="${runner_path%/*}"
  [[ "${slice_path##*/}" == "$ACE_SLICE" ]] || { fail "runner is not a direct child of ${ACE_SLICE}: $runner_path"; return 1; }
  printf '%s\n' "$slice_path"
}

check_runner_boundary() {
  local runner_path="$1"
  local slice_path
  slice_path="$(runner_slice_path "$runner_path")" || return 1
  check_slice_limits "${ACE_CGROUP_ROOT}${slice_path}"
}

check_workload_boundary() {
  local parent_path="$1"
  local child_path="$2"
  [[ "$child_path" == "${parent_path}/ace-scientific-workload-"* ]] || { fail "workload child is not directly below declared parent: $child_path"; return 1; }

  local root="${ACE_CGROUP_ROOT}${child_path}"
  [[ -d "$root" ]] || { fail "workload cgroup does not exist: $root"; return 1; }

  local cpu memory swap pids
  cpu="$(cat "$root/cpu.max")"
  memory="$(cat "$root/memory.max")"
  swap="$(cat "$root/memory.swap.max")"
  pids="$(cat "$root/pids.max")"

  [[ "$cpu" == "$ACE_CPU_MAX" ]] || { fail "unexpected workload cpu.max: $cpu"; return 1; }
  [[ "$memory" == "$ACE_MEMORY_MAX" ]] || { fail "unexpected workload memory.max: $memory"; return 1; }
  [[ "$swap" == "$ACE_SWAP_MAX" ]] || { fail "unexpected workload memory.swap.max: $swap"; return 1; }
  [[ "$pids" == "$ACE_PIDS_MAX" ]] || { fail "unexpected workload pids.max: $pids"; return 1; }

  check_slice_limits "${ACE_CGROUP_ROOT}${parent_path}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-}" in
    check-runner)
      [[ $# -eq 2 ]] || { fail "usage: $0 check-runner <runner-cgroup-path>"; exit 1; }
      check_runner_boundary "$2"
      ;;
    check-workload)
      [[ $# -eq 3 ]] || { fail "usage: $0 check-workload <parent-cgroup-path> <child-cgroup-path>"; exit 1; }
      check_workload_boundary "$2" "$3"
      ;;
    *)
      fail "usage: $0 {check-runner|check-workload} ..."
      exit 1
      ;;
  esac

  echo "ACE-CGROUP-BOUNDARY: PASS"
fi
