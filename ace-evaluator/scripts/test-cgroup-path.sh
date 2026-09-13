#!/usr/bin/env bash
set -euo pipefail

SLICE="ace-evaluator.slice"

inside_slice() {
  [[ "$1" == */${SLICE}/* ]]
}

inside_slice "/ace-evaluator.slice/actions.runner.Infrasigma-migtalk.localhost.service"
inside_slice "/ace.slice/ace-evaluator.slice/actions.runner.Infrasigma-migtalk.localhost.service"
inside_slice "/custom-parent.slice/ace.slice/ace-evaluator.slice/actions.runner.example.service"

if inside_slice "/ace.slice/other.slice/actions.runner.example.service"; then
  echo "unexpected acceptance of missing ${SLICE}" >&2
  exit 1
fi
if inside_slice "/ace.slice/ace-evaluator.slice-other/actions.runner.example.service"; then
  echo "unexpected acceptance of partial slice name" >&2
  exit 1
fi
if inside_slice "/ace.slice/not-ace-evaluator.slice/actions.runner.example.service"; then
  echo "unexpected acceptance of different slice component" >&2
  exit 1
fi

echo "cgroup path invariant tests: PASS"
