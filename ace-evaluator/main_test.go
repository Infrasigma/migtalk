package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func artifact(script string) []byte { return []byte("#!/bin/sh\n" + script + "\n") }

func hash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestEvaluateMediatesOpaqueFrames(t *testing.T) {
	capability := artifact("while IFS= read -r frame; do printf '%s\\n' '{\"kind\":\"action\",\"body\":{\"opaque\":true}}'; done")
	challenge := artifact("printf '%s\\n' '{\"kind\":\"observation\",\"body\":{\"opaque\":true}}'; IFS= read -r action; printf '%s\\n' '{\"kind\":\"result\",\"body\":{\"passed\":true}}'")
	r := request{Protocol: protocol, Plan: "plan", Capability: capability, CapabilityHash: hash(capability), Challenge: challenge, ChallengeHash: hash(challenge), WallClockNS: 2_000_000_000, InteractionBudget: 1}
	got := evaluateForTest(r)
	if got.Error != "" || !got.Passed || got.Interactions != 1 { t.Fatalf("unexpected result: %+v", got) }
}

func TestEvaluateRejectsArtifactTampering(t *testing.T) {
	capability := artifact("printf '%s\\n' '{\"kind\":\"action\"}'")
	challenge := artifact("printf '%s\\n' '{\"kind\":\"result\",\"body\":{\"passed\":true}}'")
	r := request{Protocol: protocol, Plan: "plan", Capability: capability, CapabilityHash: hash(append(capability, 'x')), Challenge: challenge, ChallengeHash: hash(challenge), WallClockNS: 1_000_000_000, InteractionBudget: 1}
	if got := evaluateForTest(r); got.Error != "capability artifact hash mismatch" { t.Fatalf("expected hash rejection, got %+v", got) }
}

func TestEvaluateEnforcesInteractionBudget(t *testing.T) {
	capability := artifact("while IFS= read -r frame; do printf '%s\\n' '{\"kind\":\"action\"}'; done")
	challenge := artifact("printf '%s\\n' '{\"kind\":\"observation\"}'; IFS= read -r action; printf '%s\\n' '{\"kind\":\"observation\"}'")
	r := request{Protocol: protocol, Plan: "plan", Capability: capability, CapabilityHash: hash(capability), Challenge: challenge, ChallengeHash: hash(challenge), WallClockNS: 2_000_000_000, InteractionBudget: 1}
	got := evaluateForTest(r)
	if got.Error != "interaction budget exceeded" { t.Fatalf("expected interaction rejection, got %+v", got) }
}
