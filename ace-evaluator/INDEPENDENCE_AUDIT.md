# ACE Evaluator Independence Audit

## Status

**REPAIRED ENGINEERING BOUNDARY — SCIENTIFIC ADMISSION STILL BLOCKED.**

This document records evidence, not self-certification. No security property is marked scientifically complete merely because source code or unit tests claim it.

## Required audit before scientific use

- [ ] evaluator has no dependency on `Infrasigma/Ace` source, binaries, generated artifacts, or learner state;
- [ ] challenge reconstruction is independently implemented;
- [ ] evaluator does not receive hidden target identity, decomposition, answer sequence, or success gradient;
- [ ] evaluator accepts only the frozen generic protocol surface;
- [ ] evaluator cannot write learner state or mutate the evaluation plan;
- [ ] evaluator emits no per-challenge result to the learner;
- [ ] stored-answer/replay attacks fail;
- [ ] representation-renaming and observation-shift attacks fail;
- [ ] target metadata/hash/order/size leakage attacks fail;
- [ ] hidden solver/template attacks fail;
- [ ] resource accounting is independently checked for every claimed dimension;
- [ ] build provenance is recorded;
- [ ] executable bytes receive a SHA-256 commitment before the scientific run;
- [ ] production-path namespace/filesystem adversarial tests pass on the exact admission host;
- [ ] cgroup-v2 CPU/memory/pid ceilings are actually active and independently audited;
- [ ] a reviewer not responsible for the learner implementation signs the semantic audit.

## Repairs implemented after adversarial review

- Deterministic target-derived sandbox naming was removed. Artifact staging uses evaluator-generated random temporary names and the sandbox exposes only `/payload` after the root transition.
- The previous direct `Chroot` launch path was replaced by a Linux user/mount/network/IPC/PID namespace child that builds a bounded tmpfs root, performs `pivot_root`, detaches `/.oldroot`, and then executes `/payload`.
- CPU/memory/pid cgroup validation is fail-closed. Missing `ACE_REQUIRE_CGROUP=1`, missing/unsafe `ACE_CGROUP_V2_PATH`, wrong cgroup membership, or unlimited `cpu.max`, `memory.max`, or `pids.max` all reject scientific execution.
- Ordinary protocol CI contains no cgroup bypass. Scientific sandbox admission is now a separate manually invoked workflow and must prove the host contract rather than pretending it exists.
- Added unit checks for fail-closed cgroup admission and target-path blindness.
- Added a production-path adversarial sandbox probe gated by `ACE_RUN_SANDBOX_TESTS=1`. It checks cwd, host filesystem, host `/tmp`, `/dev/shm`, `/proc`, hash environment leakage, and write access inside the isolated root.

## Evidence currently present

- The evaluator remains a standalone Go module under `ace-evaluator/` and imports no ACE package.
- ACE-BLIND-2 verifies exact capability and challenge artifact hashes before execution.
- Capability and challenge are launched as separate child processes and communicate only through generic observation/action/result frames.
- Wall-clock and interaction ceilings remain evaluator-enforced.
- Static ELF admission is enforced before sandbox execution.
- The ordinary GitHub Actions workflow tests, vets, builds, hashes, and uploads the evaluator without weakening the scientific contract.
- The dedicated scientific-admission workflow refuses to proceed when the requested cgroup contract is not actually active.

These are engineering facts only. They do **not** establish that the sandbox works on a particular host until the production-path adversarial test executes successfully there.

## Current blockers

1. Production-path namespace/filesystem adversarial execution has not yet been independently observed on an admission-capable Linux runner.
2. cgroup-v2 CPU/memory/pid containment has not yet been demonstrated on an actual runner with the exact frozen configuration.
3. Storage is bounded by the sandbox tmpfs capacity, but a full adversarial storage-exhaustion test remains to be run under the production sandbox.
4. Cross-subject/cross-challenge contamination still requires end-to-end adversarial execution rather than static inspection.
5. Semantic-independence and external-review requirements remain open.

**Decision: DEFER scientific execution. The evaluator is not scientifically admitted.**
