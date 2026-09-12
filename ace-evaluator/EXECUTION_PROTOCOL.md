# ACE Independent Evaluator — Executable Challenge/Capability Contract

This document fixes the evaluator-side interpretation of the neutral executable ABI without importing the ACE learner implementation.

## Two executable artifacts

The protocol now commits **both** sides of the interaction:

1. a sealed capability executable supplied by the learner harness;
2. a sealed challenge executable supplied by the evaluation harness.

Both are verified by exact SHA-256 before execution. The challenge executable owns hidden challenge semantics and its verification logic. This prevents the independent evaluator from needing ACE's environment implementation or a task-specific DSL.

Neither artifact may be interpreted as a repository-local DSL or translated into a task-specific intermediate language.

## Wire boundary

The evaluator consumes newline-delimited JSON using protocol `ACE-BLIND-2`.

Each request contains:

- plan commitment;
- capability subject and exact capability artifact bytes;
- challenge commitment;
- exact challenge artifact bytes;
- challenge artifact SHA-256.

Observation, goal, and action payloads are opaque to the evaluator. The evaluator merely mediates the separately controlled processes according to this contract.

The evaluator must not provide a success gradient, expected action, target decomposition, hidden state, target identifier, or learner feedback.

## Process rules

The evaluator must:

1. verify the exact SHA-256 of the capability artifact;
2. verify the exact SHA-256 of the challenge artifact;
3. execute capability and challenge as separately controlled child processes;
4. route only generic protocol frames between them;
5. enforce precommitted resource ceilings from outside the capability process;
6. prevent capability-local state from crossing fresh-state challenge boundaries unless persistence is explicitly being tested;
7. expose only aggregate scientific results to the learner harness;
8. fail closed on malformed frames, hash mismatch, unexpected output, process escape, or resource overrun.

## Independence boundary

The evaluator contains no ACE environment semantics. The challenge artifact is the executable semantic authority for its own hidden state, goal, dynamics, and verification. The evaluator therefore does not need to reconstruct those semantics from an ACE implementation.

## Scientific admission

This file defines transport/execution mechanics only. It is **not** an evaluator admission certificate.

Admission still requires an independently built evaluator executable, adversarial evaluator-side tests, reproducible build provenance, exact executable SHA-256, and an independent semantic audit.

No ACE source, binary, environment implementation, model implementation, learner state, or hidden ACE task semantics may be imported or reused by the evaluator.
