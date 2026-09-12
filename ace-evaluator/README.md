# ACE Independent Evaluator — Separation Boundary

This repository contains a separately maintained evaluator scaffold for the ACE sealed-capability protocol.

## Scientific admission status

NOT YET ADMITTED. This source repository is only a candidate external boundary. A scientific run requires:

1. an independently reviewed implementation;
2. a separately built executable;
3. an exact SHA-256 of that executable frozen before evaluation;
4. semantic audit confirming no import or reuse of ACE learner/environment implementation;
5. challenge reconstruction from committed challenge material without target/decomposition leakage;
6. aggregate-only output and strict resource enforcement.

Do not use this scaffold as scientific evidence merely because it builds.

## Design rule

The evaluator must implement the generic sealed-capability contract without importing `Infrasigma/Ace`, copying its environment implementation, embedding target solutions, or exposing per-challenge feedback.
