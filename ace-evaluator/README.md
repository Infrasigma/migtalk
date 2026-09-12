# ACE Independent Evaluator — Separation Boundary

This repository contains the candidate external evaluator for the ACE-BLIND-2 sealed-capability protocol.

## Scientific admission status

**NOT YET ADMITTED.** A scientific run requires:

1. an independently reviewed implementation;
2. a separately built executable;
3. an exact SHA-256 of that executable frozen before evaluation;
4. semantic audit confirming no import or reuse of ACE learner/environment implementation;
5. challenge execution from sealed challenge artifacts without target/decomposition leakage;
6. aggregate-only output and externally enforced resource limits.

Do not use this source tree or a test binary as scientific evidence merely because it builds.

## Design rule

The evaluator implements only the generic ACE-BLIND-2 process contract. It must not import `Infrasigma/Ace`, copy ACE environment code, embed target solutions, create a fixed task DSL, expose per-challenge feedback, or infer hidden challenge semantics from hashes.

The challenge executable is the semantic authority for its own hidden state, goal, dynamics, and verification. The evaluator only mediates the opaque observation/goal/action protocol and verifies artifact identity.

## Implementation gate

Before scientific admission, the implementation must demonstrate fail-closed handling of malformed frames, artifact tampering, process failure, timeout/resource exhaustion, cross-subject state leakage, replay/stored-answer attempts, and unexpected evaluator output. The build must be reproducible and produce a separately hashable executable.
