# Target identity exposure audit

## Conclusion

The previously reported challenge-filename leak is not present in the current production execution path.

`writeTempExecutable` creates an evaluator-owned random temporary host path. `newSandboxedCommand` passes only that opaque temporary path to the sandbox child. `setupSandboxFilesystem` reads the bytes from the host path and writes them into the sandbox tmpfs at the fixed path `/payload`. After `pivot_root`, the untrusted executable is launched as `/payload`.

Therefore the host filename, including any SHA-256-derived source filename, is not mounted or exposed inside the sandbox.

The child environment contains no challenge hash, and the adversarial probe already checks for `ACE_CHALLENGE_HASH` leakage and known hash-bearing challenge filenames.

## Important distinction

A historical audit claimed a path such as `challenge-<sha256>.bin` remained visible inside the sandbox. That claim does not match the actual source at the audited commit. The production path uses `/payload`, not the source filename.

We therefore do not alter correct production code merely to satisfy an inaccurate finding. This document records the source-grounded resolution and makes the property explicit for future audits.

## Scientific status

This is a static/source audit only. It is NOT evidence that the production sandbox is secure on an admission host. Kernel-level namespace and cgroup enforcement still require execution of the dedicated scientific-admission workflow on an actual compatible self-hosted Linux runner.
