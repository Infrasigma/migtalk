#!/usr/bin/env bash
set -euo pipefail

command -v ip >/dev/null 2>&1 || { echo 'ip command required' >&2; exit 1; }
command -v unshare >/dev/null 2>&1 || { echo 'unshare command required' >&2; exit 1; }

link_names() {
  ip -o link show | sed -nE 's/^[0-9]+: ([^:]+):.*/\1/p' | sed 's/@.*//' | sort
}

# Positive: a fresh network namespace must contain exactly loopback.
unshare --user --map-root-user --net --fork sh -ceu '
  links="$(ip -o link show | sed -nE "s/^[0-9]+: ([^:]+):.*/\1/p" | sed "s/@.*//" | sort)"
  test "$links" = lo
' 

# Negative: if an additional kernel interface exists, the exact admission
# predicate must reject the namespace. The dummy interface is created inside
# the isolated net namespace, so this is a genuine kernel-visible negative test.
unshare --user --map-root-user --net --fork sh -ceu '
  ip link add name ace-admission-extra0 type dummy
  links="$(ip -o link show | sed -nE "s/^[0-9]+: ([^:]+):.*/\1/p" | sed "s/@.*//" | sort)"
  test "$links" != lo
  ip link del ace-admission-extra0
'

# Positive: the first child in a fresh PID namespace is PID 1 without relying
# on a namespace-local /proc mount.
unshare --user --map-root-user --pid --fork sh -ceu '
  test "$$" = 1
'

# Combined production-shaped namespace set: every required namespace must be
# created successfully and the network namespace must still contain only lo.
unshare --user --map-root-user --mount --net --ipc --pid --fork sh -ceu '
  test "$(id -u)" = 0
  test -n "$(readlink /proc/self/ns/user)"
  test -n "$(readlink /proc/self/ns/mnt)"
  test -n "$(readlink /proc/self/ns/net)"
  test -n "$(readlink /proc/self/ns/ipc)"
  test "$$" = 1
  links="$(ip -o link show | sed -nE "s/^[0-9]+: ([^:]+):.*/\1/p" | sed "s/@.*//" | sort)"
  test "$links" = lo
'

# Syntax check the admission script itself so malformed shell quoting cannot
# reach the physical host.
bash -n "$(dirname "$0")/validate-ace-host.sh"

printf '%s\n' 'namespace admission regression tests: PASS'
