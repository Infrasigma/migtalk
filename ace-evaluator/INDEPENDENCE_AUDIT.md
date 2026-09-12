# ACE Evaluator Independence Audit

## Status

CANDIDATE BOUNDARY — NOT SCIENTIFICALLY ADMITTED.

## Required audit before use

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
- [ ] resource accounting is independently checked;
- [ ] build provenance is recorded;
- [ ] executable bytes receive a SHA-256 commitment before the scientific run;
- [ ] a reviewer not responsible for the learner implementation signs the semantic audit.

A checklist completion is not itself proof. Each item requires concrete evidence and preserved artifacts.
