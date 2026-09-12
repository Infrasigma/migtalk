# ACE Evaluator Repair Audit — 2026-09-12

## Audited base

Branch: `ace-independent-evaluator-20260912`

Pre-repair audit commit: `3326abd5303250507e9c5a11d2247af94e0a6805`

## Important source-vs-report reconciliation

The pre-repair report referenced `sandbox/linux_namespaces.go`, `sandbox/cgroup_manager.go`, and an `old_root_fd` leak. The actual branch stores the implementation under `ace-evaluator/`, and the inspected `filesystem_linux.go` performs `pivot_root` without retaining an explicit old-root file descriptor. Go's `os/exec` child creation also does not inherit arbitrary parent descriptors through `ExtraFiles` when none are supplied. Therefore the claimed `O_CLOEXEC` defect was not reproduced from the actual source and was not patched as though it existed.

The actual source did have material containment weaknesses that were repaired below.

## Repairs applied

1. Bounded tmpfs storage was tightened to 16 MiB and `nr_inodes=1024` was added.
2. Tmpfs remains executable because `/payload` is the untrusted executable; an attempted `noexec` hardening change was reverted before verification because it would make the sandbox unable to execute the payload.
3. Scientific cgroup admission now requires finite `cpu.max`, `memory.max`, `pids.max`, and `memory.swap.max`.
4. `memory.swap.max` must equal `0`; otherwise scientific admission fails closed.
5. Scientific admission workflow now targets an explicitly provisioned self-hosted Linux runner labeled `ace-cgroup-v2` rather than a standard hosted runner.
6. The workflow explicitly checks the requested cgroup files and verifies `memory.swap.max == 0` before production-path tests.
7. Production-path sandbox adversarial testing now includes an inode-exhaustion probe and requires the bounded inode ceiling to stop file creation before host-level exhaustion.
8. Existing artifact staging already uses evaluator-generated temporary names and exposes `/payload` after the root transition; no target hash appears in the child argv/environment from the inspected implementation.

## Verification state

No scientific ACE experiment was executed.

No scientific result was generated.

GitHub reports no status records for the latest repair commit at the time of this audit, so CI is **NOT CLAIMED PASS**.

The production sandbox admission workflow is intentionally manual and requires an actual admission-capable self-hosted runner. A standard GitHub-hosted runner is not treated as scientific evidence.

## Remaining admission requirement

Before scientific use, run the production-path adversarial workflow on a provisioned Linux runner satisfying the frozen cgroup-v2 contract and independently inspect the resulting containment evidence.
