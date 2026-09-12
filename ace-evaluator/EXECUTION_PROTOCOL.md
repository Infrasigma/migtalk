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

Scientific resource accounting is restricted to quantities that are externally enforceable:

- wall-clock time;
- interaction count;
- sealed artifact size;
- evaluator-mediated frame/output size.

CPU and memory are **not** accepted as self-reported telemetry. A scientific host must place the evaluator itself in a precommitted cgroup v2 with finite CPU and memory ceilings. The evaluator verifies that it is actually executing inside that cgroup and fails closed otherwise. The cgroup configuration is part of the frozen scientific execution environment and must be independently audited.

Search effort, disk I/O, GPU/TPU use, and internal adaptation remain outside this evaluator's independently measured resource vector unless a separately audited host layer is added. The sandbox itself uses a finite tmpfs root, so local filesystem capacity is host-enforced within the sandbox, but this is not a claim about a global disk-I/O budget.

## Process isolation requirements

Every capability/challenge execution must use a fresh Linux sandbox with:

- user namespace with a root mapping only to the invoking evaluator UID/GID;
- private mount namespace with private propagation;
- network namespace;
- IPC namespace;
- PID namespace;
- private process group and parent-death termination;
- bounded tmpfs execution root;
- `pivot_root` followed by detachment of the old root;
- no inherited environment beyond a minimal fixed environment;
- statically linked ELF executable only.

The sandbox contains only the sealed executable at the fixed path `/payload`. The capability and challenge run in distinct fresh sandboxes. There is no shared host `/tmp`, `/dev/shm`, user home, or arbitrary host filesystem after the root transition, and no shared IPC namespace. Each execution receives a distinct mount namespace and bounded ephemeral root.

The evaluator must fail closed if the host cannot create the required namespaces, root transition, bounded filesystem, or cgroup contract.

## Hash blindness

Artifact SHA-256 values are used only for commitment verification by the evaluator. The evaluator must never pass capability or challenge hashes, artifact hashes, target IDs, challenge ordering metadata, or other identifying metadata into child arguments, environment variables, filenames, mountpoints, or generic protocol frames.

The evaluator may use random opaque host-side temporary names solely to stage sealed artifacts before namespace entry. Those names must contain no deterministic function of the challenge, family, target, plan, or artifact hash. Inside the sandbox the sealed executable is exposed only at `/payload`.

## Process rules

The evaluator must:

1. verify exact SHA-256 of the capability artifact;
2. verify exact SHA-256 of the challenge artifact;
3. reject non-ELF or dynamically linked artifacts;
4. execute capability and challenge as separately controlled sandboxed child processes;
5. isolate their process groups and clean descendants on termination;
6. route only generic protocol frames between them;
7. enforce precommitted wall-clock and interaction ceilings externally;
8. bound artifact and mediated-output sizes externally;
9. prevent capability-local state from crossing fresh challenge boundaries unless persistence is explicitly being tested;
10. expose only aggregate scientific results to the learner harness;
11. fail closed on malformed frames, hash mismatch, unexpected output, process failure/escape, namespace failure, filesystem failure, cgroup failure, or resource overrun.

## Independence boundary

The evaluator contains no ACE environment semantics. The challenge artifact is the executable semantic authority for its hidden state, goal, dynamics, and verification. The evaluator therefore does not reconstruct those semantics from ACE source.

## Scientific admission

This file defines transport/execution mechanics only. It is **not** an evaluator admission certificate.

Admission still requires:

- successful reproducible build;
- exact frozen evaluator SHA-256;
- evaluator-side adversarial tests that exercise the production sandbox path;
- concrete semantic-independence audit;
- independently audited Linux namespace/filesystem and cgroup configuration;
- review/approval by an auditor who did not implement the ACE learner/environment.

No ACE source, binary, environment implementation, model implementation, learner state, or hidden ACE task semantics may be imported or reused by the evaluator.
