# ACE Independent Evaluator — Executable Challenge/Capability Contract

This document fixes the evaluator-side interpretation of the neutral executable ABI without importing the ACE learner implementation.

## Two executable artifacts

The protocol commits both sides of the interaction:

1. a sealed capability executable supplied by the learner harness;
2. a sealed challenge executable supplied by the evaluation harness.

Both are verified by exact SHA-256 before execution. The challenge executable owns hidden challenge semantics and verification logic. The evaluator never needs ACE's environment implementation or a task-specific DSL.

Neither artifact may be interpreted as a repository-local DSL or translated into a task-specific intermediate language.

## Wire boundary

The evaluator consumes newline-delimited JSON using protocol `ACE-BLIND-2`.

Each request contains the plan commitment, exact capability artifact/hash, exact challenge artifact/hash, and only the resource limits that the evaluator can enforce independently.

Observation, goal, and action payloads are opaque to the evaluator. The evaluator merely mediates the separately controlled processes according to this contract.

The evaluator must not provide a success gradient, expected action, target decomposition, hidden state, target identifier, or learner feedback.

## Independently enforceable resources

Scientific resource accounting is restricted to quantities the evaluator can enforce from outside the child processes:

- wall-clock time;
- interaction count;
- sealed artifact size;
- evaluator-mediated frame/output size.

The evaluator must **not** claim independent enforcement of CPU instructions, memory residency, storage/search effort, or internal adaptation work from self-reported child telemetry. Those dimensions require a separate host-level instrumentation/sandbox contract and are therefore outside this executable evaluator's admission claim.

A scientific run may use only the enforceable resource vector unless an independently audited host instrumentation layer is added and frozen separately.

## Process rules

The evaluator must:

1. verify exact SHA-256 of the capability artifact;
2. verify exact SHA-256 of the challenge artifact;
3. execute capability and challenge as separately controlled child processes;
4. isolate their process groups and clean descendants on termination;
5. route only generic protocol frames between them;
6. enforce precommitted wall-clock and interaction ceilings externally;
7. bound artifact and mediated-output sizes externally;
8. prevent capability-local state from crossing fresh challenge boundaries unless persistence is explicitly being tested;
9. expose only aggregate scientific results to the learner harness;
10. fail closed on malformed frames, hash mismatch, unexpected output, process failure/escape, or resource overrun.

## Independence boundary

The evaluator contains no ACE environment semantics. The challenge artifact is the executable semantic authority for its hidden state, goal, dynamics, and verification. The evaluator therefore does not reconstruct those semantics from ACE source.

## Scientific admission

This file defines transport/execution mechanics only. It is **not** an evaluator admission certificate.

Admission still requires:

- successful reproducible build;
- exact frozen evaluator SHA-256;
- evaluator-side adversarial tests;
- concrete semantic-independence audit;
- review/approval by an auditor who did not implement the ACE learner/environment.

No ACE source, binary, environment implementation, model implementation, learner state, or hidden ACE task semantics may be imported or reused by the evaluator.
