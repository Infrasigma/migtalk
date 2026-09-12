# ACE Evaluator Independence Audit

## Status

**ENGINEERING BOUNDARY IMPLEMENTED — SCIENTIFIC ADMISSION BLOCKED.**

This document records evidence, not a self-certification. No item is marked complete merely because the implementation claims it.

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
- [ ] a reviewer not responsible for the learner implementation signs the semantic audit.

## Evidence currently present

- The evaluator is implemented as a standalone Go module under `ace-evaluator/` and imports no ACE package.
- ACE-BLIND-2 verifies exact capability and challenge artifact hashes before execution.
- Capability and challenge are launched as separate child processes and communicate only through generic observation/action/result frames.
- Repository-side tests cover successful opaque-frame mediation, artifact tampering rejection, and interaction-budget exhaustion.
- GitHub Actions workflow is committed to test, vet, build, and hash the standalone Linux evaluator.

These are engineering facts only. They do **not** establish semantic independence, adversarial completeness, sandbox/resource equivalence, or external review.

## Remaining blockers

1. A successful CI build must produce a retrievable standalone evaluator artifact and frozen SHA-256.
2. The evaluator must gain independent enforcement of every resource dimension claimed by the ACE protocol, or those dimensions must be removed from the scientific claim.
3. Child-process isolation must be audited for process escape, filesystem/network access, and descendant-process cleanup; `exec.CommandContext` plus a timeout is not a complete sandbox.
4. Evaluator-side adversarial tests must cover replay/stored answers, cross-subject contamination, metadata leakage, representation shifts, malformed output, and hidden-solver behavior.
5. A reviewer who did not implement the ACE learner must perform and sign the semantic independence audit.

**Decision: DEFER scientific execution. Do not run the ACE capability-compounding experiment from this branch until the blockers above are genuinely closed.**
