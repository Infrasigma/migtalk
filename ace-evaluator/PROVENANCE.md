# ACE Evaluator Provenance Record

## Status

CANDIDATE EXTERNAL BOUNDARY — NOT SCIENTIFICALLY ADMITTED.

This record exists to make provenance explicit before any evaluator implementation is admitted as scientific infrastructure.

## Repository origin

- Repository: `Infrasigma/migtalk`
- Repository type: GitHub fork
- Parent/source repository: `DietrichGebert/ponytail`
- Parent/source owner: `DietrichGebert`
- ACE learner repository: `Infrasigma/Ace`
- Relationship to ACE: separate repository; this repository did not originate as an ACE fork.

The GitHub repository metadata identifies `Infrasigma/migtalk` as a fork and identifies `DietrichGebert/ponytail` as its parent/source. This is provenance evidence only; it is not proof of semantic independence.

## Evaluator-branch boundary

Branch: `ace-independent-evaluator-20260912`

The ACE evaluator material was added after the fork existed. The evaluator must remain isolated from ACE implementation details. In particular, the scientific evaluator must not import, vendor, copy, invoke, or derive hidden semantics from `Infrasigma/Ace` learner/environment code.

## Admission requirements

Provenance alone is insufficient. Before scientific use, preserve evidence for all of the following:

1. no ACE source/binary/generated-artifact dependency;
2. independent challenge reconstruction from frozen generic challenge material;
3. no target identity, decomposition, answer sequence, success gradient, or learner feedback leakage;
4. independent evaluator-side semantic implementation;
5. evaluator-specific adversarial tests;
6. separately built executable and exact SHA-256 commitment;
7. host-enforced resource boundary;
8. semantic review by a reviewer not responsible for the learner implementation.

Until these requirements are evidenced, the evaluator remains a candidate boundary and must not be used to claim scientific results.
