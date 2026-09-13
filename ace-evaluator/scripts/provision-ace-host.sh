#!/usr/bin/env bash
set -euo pipefail

# Provisions the host contract required by the ACE scientific admission job.
# This script does NOT register a GitHub runner or mint credentials. Registration
# remains an explicit operator action because GitHub runner tokens are short-lived.
# Run on a dedicated Linux host with systemd as PID 1.

SLICE="ace-evaluator.slice"
SLICE_UNIT="/etc/systemd/system/${SLICE}"
DROPIN_ROOT="/etc/systemd/system"
RUNNER_SERVICE="${ACE_RUNNER_SERVICE:-}"

fail() { echo "ACE-HOST-BOOTSTRAP: FAIL: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"; }

[[ "$(uname -s)" == "Linux" ]] || fail "Linux required"
[[ "$(ps -p 1 -o comm=)" == "systemd" ]] || fail "systemd PID 1 required"
[[ $EUID -eq 0 ]] || fail "run as root"
need systemctl
need awk
need findmnt
need unshare
need runuser

CGROUP2="$(findmnt -n -o FSTYPE /sys/fs/cgroup 2>/dev/null || true)"
[[ "$CGROUP2" == "cgroup2" ]] || fail "/sys/fs/cgroup is not cgroup v2"

CONTROLLERS="$(cat /sys/fs/cgroup/cgroup.controllers 2>/dev/null || true)"
for c in cpu memory pids; do
  grep -qw "$c" <<<"$CONTROLLERS" || fail "cgroup v2 controller unavailable: $c"
done

cat >"$SLICE_UNIT" <<'EOF'
[Unit]
Description=ACE scientific evaluator resource boundary
Before=multi-user.target

[Slice]
CPUAccounting=yes
MemoryAccounting=yes
TasksAccounting=yes
CPUQuota=100%
MemoryMax=1G
MemorySwapMax=0
TasksMax=256
EOF

systemctl daemon-reload
systemctl start "$SLICE"

if [[ -z "$RUNNER_SERVICE" ]]; then
  mapfile -t services < <(systemctl list-unit-files 'actions.runner.*.service' --no-legend --no-pager | awk '{print $1}')
  if (( ${#services[@]} != 1 )); then
    fail "expected exactly one installed actions.runner.*.service; found ${#services[@]}. Set ACE_RUNNER_SERVICE explicitly."
  fi
  RUNNER_SERVICE="${services[0]}"
fi
systemctl cat "$RUNNER_SERVICE" >/dev/null 2>&1 || fail "runner service not found: $RUNNER_SERVICE"
RUNNER_USER="$(systemctl show -p User --value "$RUNNER_SERVICE")"
[[ -n "$RUNNER_USER" ]] || RUNNER_USER=root

DROPIN_DIR="${DROPIN_ROOT}/${RUNNER_SERVICE}.d"
mkdir -p "$DROPIN_DIR"
cat >"${DROPIN_DIR}/10-ace-evaluator.conf" <<EOF
[Service]
Slice=${SLICE}
Delegate=yes
EOF

systemctl daemon-reload
systemctl restart "$RUNNER_SERVICE"
sleep 2
systemctl is-active --quiet "$RUNNER_SERVICE" || fail "runner service failed after cgroup placement"

# Test the actual runner identity, not root. This catches host policies that
# permit namespace creation for root while denying it to the service account.
runuser -u "$RUNNER_USER" -- unshare --user --map-root-user --mount --net --ipc --pid --fork true \
  || fail "runner identity ${RUNNER_USER} cannot create required namespaces"

CGROUP_PATH="$(systemctl show -p ControlGroup --value "$RUNNER_SERVICE")"
SYSTEMD_SLICE="$(systemctl show -p Slice --value "$RUNNER_SERVICE")"
[[ "$SYSTEMD_SLICE" == "$SLICE" ]] || fail "runner is not assigned to ${SLICE}: ${SYSTEMD_SLICE}"
[[ "$CGROUP_PATH" == */${SLICE}/* ]] || fail "runner is not inside ${SLICE}: ${CGROUP_PATH}"

# cgroup-v2 limits are hierarchical. The runner service cgroup may legitimately
# read `max` because it delegates child management; the enforced resource cap is
# defined by the ACE slice above it. Check that enforcing ancestor directly.
source "$(dirname "$0")/check-cgroup-boundary.sh"
ACE_SLICE_PATH="$(runner_slice_path "$CGROUP_PATH")"
check_slice_limits "/sys/fs/cgroup${ACE_SLICE_PATH}" || fail "invalid ACE enforcing slice"

ROOT="/sys/fs/cgroup${CGROUP_PATH}"
[[ -r "$ROOT/cpu.max" ]] || fail "missing cpu.max at $ROOT"
[[ -r "$ROOT/memory.max" ]] || fail "missing memory.max at $ROOT"
[[ -r "$ROOT/memory.swap.max" ]] || fail "missing memory.swap.max at $ROOT"
[[ -r "$ROOT/pids.max" ]] || fail "missing pids.max at $ROOT"
[[ -w "$ROOT/cgroup.procs" ]] || fail "runner cgroup is not writable by delegated runner service"
[[ -w "$ROOT/cgroup.subtree_control" ]] || fail "runner cgroup does not expose delegated controller management"

cat <<EOF
ACE-HOST-BOOTSTRAP: PASS
runner_service=${RUNNER_SERVICE}
runner_user=${RUNNER_USER}
cgroup_path=${CGROUP_PATH}
cgroup_root=${ROOT}
ace_slice_path=${ACE_SLICE_PATH}
ace_slice_cpu.max=$(cat "/sys/fs/cgroup${ACE_SLICE_PATH}/cpu.max")
ace_slice_memory.max=$(cat "/sys/fs/cgroup${ACE_SLICE_PATH}/memory.max")
ace_slice_memory.swap.max=$(cat "/sys/fs/cgroup${ACE_SLICE_PATH}/memory.swap.max")
ace_slice_pids.max=$(cat "/sys/fs/cgroup${ACE_SLICE_PATH}/pids.max")
runner_cpu.max=$(cat "$ROOT/cpu.max")
runner_memory.max=$(cat "$ROOT/memory.max")
runner_memory.swap.max=$(cat "$ROOT/memory.swap.max")
runner_pids.max=$(cat "$ROOT/pids.max")
delegate=yes
workload_boundary=delegated-child-cgroup
next=run ace-evaluator/scripts/validate-ace-host.sh ${CGROUP_PATH}
EOF
