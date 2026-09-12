# ACE physical validation route

This document is an operational bridge from source-correct to kernel-enforced and adversarially demonstrated. It does not certify the evaluator by itself.

## 1. Required host

Use a dedicated Linux VM or bare-metal Linux host with systemd and cgroup v2. A self-hosted GitHub Actions runner is the preferred reproducible CI route. GitHub-hosted runners are deliberately excluded from the scientific gate because the admission contract requires the runner process to be placed in a known finite cgroup hierarchy.

The host must permit an unprivileged process to create a user namespace. The production sandbox then creates the required mount, network, IPC, and PID namespaces in the same clone operation. The host probe exercises the real kernel rather than testing configuration files only.

## 2. Provision

From a checkout of the exact evaluator branch, as root:

```sh
sudo ACE_RUNNER_SERVICE=actions.runner.<owner>-<repo>.<runner>.service \
  ./ace-evaluator/scripts/provision-ace-host.sh
```

The script creates `ace-evaluator.slice` with finite CPU, memory, swap, and PID ceilings, places the GitHub runner service into that slice, and refuses to continue if the kernel cannot create the required namespaces.

Runner registration is intentionally not automated: GitHub registration tokens are short-lived credentials and must be obtained through the repository's **Settings → Actions → Runners → New self-hosted runner** flow. After registration, install the runner as its systemd service and then run the provisioning script. GitHub documents both service installation and custom labels for self-hosted runners.

The runner must have the custom label `ace-cgroup-v2` used by the scientific workflow.

## 3. Machine admission

The provisioning script prints the runner service's exact cgroup path. Run:

```sh
./ace-evaluator/scripts/validate-ace-host.sh /ace-evaluator.slice/<runner-service>.service
```

The probe records kernel/OS/cgroup/mount/namespace facts, tests user+mount+network+IPC+PID namespace creation, tests bounded tmpfs and `pivot_root` in a disposable namespace, verifies that the current process is in the declared cgroup, and writes a SHA-256 evidence manifest.

## 4. Production-path admission

Dispatch `.github/workflows/ace-sandbox-admission.yml` on the `ace-cgroup-v2` runner and supply the exact cgroup path printed by the host bootstrap. The workflow:

1. verifies cgroup v2 and finite CPU/memory/PID ceilings;
2. requires `memory.swap.max=0`;
3. runs the machine-verifiable kernel admission probe;
4. builds the exact evaluator executable and records its SHA-256;
5. executes the production sandbox path, not a mock;
6. uploads the evidence bundle keyed to the GitHub run ID.

A green workflow is still only a candidate admission result until the evidence bundle is independently inspected against the frozen source SHA and the semantic-independence review.

## 5. Evidence vocabulary

Every protection must be classified separately:

- `SOURCE-CORRECT`: static source audit establishes the intended behavior.
- `KERNEL-ENFORCED`: the physical host probe establishes that the required Linux primitive exists and is active.
- `ADVERSARIAL-PROBE-PASSED`: the production sandbox execution demonstrates the boundary under attack.

Never convert the first category into either of the latter two by inference.

## 6. No false admission

If no compatible runner is available, the correct result is `DEFERRED — PHYSICAL SANDBOX VALIDATION REQUIRED`. Do not change the workflow to run on `ubuntu-latest`, replace namespace/cgroup tests with mocks, or turn a failed probe into a warning. The bootstrap and evidence machinery exist specifically so the remaining external resource requirement is explicit and reproducible.
