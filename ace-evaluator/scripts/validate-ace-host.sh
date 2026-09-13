#!/usr/bin/env bash
set -euo pipefail

# Read-only admission probe. It never substitutes for the production sandbox
# test. It proves the host can perform the kernel operations that production
# depends on, and emits a cryptographically hashable evidence bundle.

CGROUP_PATH="${1:-${ACE_CGROUP_V2_PATH:-}}"
OUT="${ACE_EVIDENCE_DIR:-ace-evidence-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"

fail() { echo "ACE-HOST-VALIDATION: FAIL: $*" >&2; exit 1; }
record() { printf '%s\n' "$*" | tee -a "$OUT/validation.log" >/dev/null; }
need() { command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"; }

[[ "$(uname -s)" == Linux ]] || fail "Linux required"
[[ -n "$CGROUP_PATH" && "$CGROUP_PATH" == /* && "$CGROUP_PATH" != *..* ]] || fail "absolute cgroup path required"
need findmnt
need unshare
need mount
need pivot_root
need sha256sum
need ip

ROOT="/sys/fs/cgroup${CGROUP_PATH}"
[[ -d "$ROOT" ]] || fail "cgroup does not exist: $ROOT"
[[ "$(findmnt -n -o FSTYPE /sys/fs/cgroup)" == cgroup2 ]] || fail "cgroup v2 not mounted"

# The runner service may expose `max` at its leaf cgroup because it delegates
# child creation. The scientific resource contract is enforced by its direct
# ace-evaluator.slice ancestor, so validate that ancestor rather than the leaf.
source "$(dirname "$0")/check-cgroup-boundary.sh"
ACE_SLICE_PATH="$(runner_slice_path "$CGROUP_PATH")"
check_slice_limits "/sys/fs/cgroup${ACE_SLICE_PATH}" || fail "invalid ACE enforcing slice"

for f in cpu.max memory.max memory.swap.max pids.max cgroup.procs; do
  [[ -r "$ROOT/$f" ]] || fail "missing $ROOT/$f"
done
[[ -w "$ROOT/cgroup.procs" ]] || fail "runner cgroup is not writable by delegated runner service"
[[ -w "$ROOT/cgroup.subtree_control" ]] || fail "runner cgroup does not expose delegated controller management"

# Capture immutable host facts before running kernel probes.
uname -a >"$OUT/uname.txt"
cat /etc/os-release >"$OUT/os-release.txt"
cat /proc/cmdline >"$OUT/kernel-cmdline.txt"
cat /proc/self/cgroup >"$OUT/self-cgroup.txt"
cat /proc/self/mountinfo >"$OUT/mountinfo.txt"
cat /proc/self/status >"$OUT/self-status.txt"
cat /sys/fs/cgroup/cgroup.controllers >"$OUT/root-cgroup-controllers.txt"
cat "$ROOT/cgroup.controllers" >"$OUT/cgroup-controllers.txt"
cat "$ROOT/cgroup.subtree_control" >"$OUT/cgroup-subtree-control.txt"
for f in cpu.max memory.max memory.swap.max pids.max; do cat "$ROOT/$f" >"$OUT/$f.txt"; done
for f in /proc/sys/user/max_user_namespaces /proc/sys/kernel/unprivileged_userns_clone; do
  if [[ -r "$f" ]]; then cat "$f" >>"$OUT/sysctls.txt"; fi
done

# Capture the actual enforcing ACE slice limits separately from the runner leaf.
for f in cpu.max memory.max memory.swap.max pids.max; do
  cat "/sys/fs/cgroup${ACE_SLICE_PATH}/$f" >"$OUT/ace-slice-$f.txt"
done
printf '%s\n' "$ACE_SLICE_PATH" >"$OUT/ace-slice-path.txt"

# Bind the evidence to the exact GitHub Actions execution when the workflow
# supplies these values. They are metadata only; none is exposed to the agent.
{
  printf 'GITHUB_RUN_ID=%s\n' "${GITHUB_RUN_ID:-UNSET}"
  printf 'GITHUB_RUN_ATTEMPT=%s\n' "${GITHUB_RUN_ATTEMPT:-UNSET}"
  printf 'GITHUB_SHA=%s\n' "${GITHUB_SHA:-UNSET}"
  printf 'GITHUB_JOB=%s\n' "${GITHUB_JOB:-UNSET}"
  printf 'GITHUB_WORKFLOW=%s\n' "${GITHUB_WORKFLOW:-UNSET}"
  printf 'GITHUB_REF=%s\n' "${GITHUB_REF:-UNSET}"
  printf 'GITHUB_REPOSITORY=%s\n' "${GITHUB_REPOSITORY:-UNSET}"
  printf 'UTC_START=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} >"$OUT/workflow-metadata.txt"

# Real kernel namespace admission. User namespaces are intentionally created
# together with the dependent namespaces; this is the same privilege model
# used by the production Go Cloneflags path.
unshare --user --map-root-user --mount --net --ipc --pid --fork sh -ceu '
  test "$(id -u)" = 0
  test -n "$(readlink /proc/self/ns/user)"
  test -n "$(readlink /proc/self/ns/mnt)"
  test -n "$(readlink /proc/self/ns/net)"
  test -n "$(readlink /proc/self/ns/ipc)"
  test "$$" = 1
  test "$(ip -o link show | sed -nE "s/^[0-9]+: ([^:]+):.*/\1/p" | sed "s/@.*//" | sort)" = lo
' >"$OUT/namespace-probe.txt" 2>&1 \
  || fail "required namespace probe failed"

# Real mount + bounded tmpfs + pivot_root probe. This is a disposable nested
# namespace; it does not alter the host mount table.
unshare --user --map-root-user --mount --net --ipc --pid --fork sh -ceu '
  mount --make-rprivate /
  d=$(mktemp -d)
  mount -t tmpfs -o size=16m,nr_inodes=1024,nosuid,nodev tmpfs "$d"
  printf x >"$d/payload"
  mkdir "$d/oldroot"
  mount --bind "$d" "$d"
  cd "$d"
  pivot_root . oldroot
  cd /
  test -f /payload
  test -f /oldroot/etc/passwd
  umount -l /oldroot
  test ! -e /oldroot/etc/passwd
  rm -f /payload
' >"$OUT/mount-pivot-probe.txt" 2>&1 \
  || fail "mount/pivot_root/tmpfs probe failed"

# Network namespace must be distinct and have no externally configured link.
unshare --user --map-root-user --net --fork sh -ceu '
  test "$(ip -o link show | sed -nE "s/^[0-9]+: ([^:]+):.*/\1/p" | sed "s/@.*//" | sort)" = lo
' >"$OUT/network-probe.txt" 2>&1 \
  || fail "network namespace probe failed"

# IPC namespace identity must differ from the host while remaining usable.
host_ipc="$(readlink /proc/self/ns/ipc)"
inner_ipc="$(unshare --user --map-root-user --ipc --fork sh -c 'readlink /proc/self/ns/ipc')"
[[ "$host_ipc" != "$inner_ipc" ]] || fail "IPC namespace was not isolated"
printf 'host=%s\ninner=%s\n' "$host_ipc" "$inner_ipc" >"$OUT/ipc-probe.txt"

# PID namespace must give the first child PID 1. /proc itself is intentionally
# not mounted here; PID identity is proved directly from the new namespace.
unshare --user --map-root-user --pid --fork sh -ceu '
  test "$$" = 1
' >"$OUT/pid-probe.txt" 2>&1 \
  || fail "PID namespace probe failed"

# Verify that the current runner process really is the process in the declared
# scientific runner cgroup. This prevents passing a sibling/parent cgroup by mistake.
grep -Fqx "0::${CGROUP_PATH}" /proc/self/cgroup \
  || fail "current process is not in requested cgroup: $(cat /proc/self/cgroup)"

# Freeze exact repository/probe provenance when available.
if command -v git >/dev/null 2>&1 && git rev-parse --show-toplevel >/dev/null 2>&1; then
  git rev-parse HEAD >"$OUT/git-head.txt"
  git status --porcelain=v1 >"$OUT/git-status.txt"
  git rev-parse HEAD:ace-evaluator >"$OUT/evaluator-tree.txt" 2>/dev/null || true
  git ls-files ace-evaluator | sort | xargs -r sha256sum >"$OUT/evaluator-source-sha256.txt"
fi

printf 'UTC_END=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$OUT/workflow-metadata.txt"

sha256sum "$OUT"/* >"$OUT/SHA256SUMS"
cat <<EOF | tee "$OUT/ADMISSION.txt"
ACE_HOST_ADMISSION=PASS
CGROUP_PATH=${CGROUP_PATH}
ACE_SLICE_PATH=${ACE_SLICE_PATH}
CGROUP_ROOT=${ROOT}
ACE_SLICE_ROOT=/sys/fs/cgroup${ACE_SLICE_PATH}
EVIDENCE_DIR=${OUT}
EVIDENCE_SHA256=$(sha256sum "$OUT/SHA256SUMS" | awk '{print $1}')
NOTE=This is host/kernel admission evidence only; scientific evaluator admission additionally requires production-path adversarial execution and independent audit.
EOF

record "ACE-HOST-VALIDATION: PASS"
record "evidence=$OUT"
record "evidence_sha256=$(sha256sum "$OUT/SHA256SUMS" | awk '{print $1}')"
