# ACE Independent Evaluator — Executable Capability Contract

This document fixes the evaluator-side interpretation of the neutral executable ABI without importing the ACE learner implementation.

## Capability artifact

The sealed artifact is an executable byte sequence. The evaluator must verify its SHA-256 against the sealed artifact commitment before execution.

The evaluator must execute the artifact as a separate child process. It must not interpret the artifact as a repo-local DSL, inspect its internal model representation, or translate it into a task-specific intermediate language.

## Wire boundary

The evaluator and capability exchange newline-delimited JSON messages using the committed execution protocol version `stdin-stdout-json-v1`.

The semantic fields are opaque byte payloads at this boundary:

- observation: sequence number, payload, terminal flag;
- goal: payload;
- action: payload plus termination flag.

The evaluator supplies only the observation and goal required by the frozen challenge. It does not send a success gradient, expected action, target decomposition, hidden state, target identifier, or learner feedback.

## Process rules

The evaluator must:

1. materialize only the hash-committed artifact bytes;
2. execute exactly those bytes in a separately controlled process;
3. enforce the precommitted resource ceilings from outside the capability process;
4. discard capability-local state between independently defined fresh-state challenges unless persistence is explicitly being tested;
5. record per-challenge outcomes internally but expose only the aggregate protocol result to the learner harness;
6. fail closed on malformed frames, artifact hash mismatch, unexpected output, process escape, or resource overrun.

## Scientific boundary

This file defines transport/execution mechanics only. It is **not** an evaluator admission certificate.

Scientific admission additionally requires independently reconstructed challenge semantics, adversarial tests for leakage/replay/stored answers/representation tricks/resource lies, a separately built executable with frozen SHA-256, and an independent semantic audit.

No ACE source, binary, environment implementation, model implementation, or learner state may be imported or reused by the evaluator.
