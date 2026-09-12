package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const protocol = "ACE-BLIND-2"

type frame struct {
	Kind string          `json:"kind"`
	Body json.RawMessage `json:"body,omitempty"`
}

type request struct {
	Protocol string `json:"protocol_version"`
	Plan string `json:"plan_commitment"`
	Capability []byte `json:"capability_artifact"`
	CapabilityHash string `json:"capability_artifact_sha256"`
	Challenge []byte `json:"challenge_artifact"`
	ChallengeHash string `json:"challenge_artifact_sha256"`
	WallClockNS uint64 `json:"wall_clock_ns"`
	InteractionBudget uint64 `json:"interaction_budget"`
}

type response struct {
	Passed bool `json:"passed"`
	Interactions uint64 `json:"interactions"`
	WallClockNS uint64 `json:"wall_clock_ns"`
	Error string `json:"error,omitempty"`
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validHash(got, want string) bool { return len(want) == 64 && strings.EqualFold(got, want) }

func writeTempExecutable(b []byte) (string, error) {
	f, err := os.CreateTemp("", "ace-evaluator-artifact-*")
	if err != nil { return "", err }
	path := f.Name()
	defer func() { _ = f.Close() }()
	if _, err := f.Write(b); err != nil { _ = os.Remove(path); return "", err }
	if err := f.Close(); err != nil { _ = os.Remove(path); return "", err }
	if err := os.Chmod(path, 0700); err != nil { _ = os.Remove(path); return "", err }
	return path, nil
}

// runInteraction starts fresh capability and challenge processes for one
// challenge. The evaluator interprets only frame kinds; observation, goal,
// and action bodies remain opaque. The challenge executable is the semantic
// authority and initiates each episode by emitting an observation frame.
func runInteraction(ctx context.Context, r request) (bool, uint64, error) {
	capPath, err := writeTempExecutable(r.Capability)
	if err != nil { return false, 0, err }
	defer os.Remove(capPath)
	chalPath, err := writeTempExecutable(r.Challenge)
	if err != nil { return false, 0, err }
	defer os.Remove(chalPath)

	capCmd := exec.CommandContext(ctx, capPath)
	chalCmd := exec.CommandContext(ctx, chalPath)
	capIn, err := capCmd.StdinPipe(); if err != nil { return false, 0, err }
	capOut, err := capCmd.StdoutPipe(); if err != nil { return false, 0, err }
	chalIn, err := chalCmd.StdinPipe(); if err != nil { return false, 0, err }
	chalOut, err := chalCmd.StdoutPipe(); if err != nil { return false, 0, err }
	capCmd.Stderr = io.Discard; chalCmd.Stderr = io.Discard
	if err := capCmd.Start(); err != nil { return false, 0, fmt.Errorf("start capability: %w", err) }
	if err := chalCmd.Start(); err != nil { _ = capCmd.Process.Kill(); _ = capCmd.Wait(); return false, 0, fmt.Errorf("start challenge: %w", err) }
	defer func() { _ = capIn.Close(); _ = chalIn.Close(); _ = capCmd.Process.Kill(); _ = chalCmd.Process.Kill(); _ = capCmd.Wait(); _ = chalCmd.Wait() }()

	capDec := json.NewDecoder(bufio.NewReader(capOut)); capEnc := json.NewEncoder(capIn)
	chalDec := json.NewDecoder(bufio.NewReader(chalOut)); chalEnc := json.NewEncoder(chalIn)
	var interactions uint64

	for {
		var cf frame
		if err := chalDec.Decode(&cf); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) { return false, interactions, errors.New("wall-clock budget exceeded") }
			return false, interactions, fmt.Errorf("challenge frame: %w", err)
		}
		switch cf.Kind {
		case "observation":
			if r.InteractionBudget > 0 && interactions >= r.InteractionBudget { return false, interactions, errors.New("interaction budget exceeded") }
			if err := capEnc.Encode(cf); err != nil { return false, interactions, fmt.Errorf("forward observation: %w", err) }
			var af frame
			if err := capDec.Decode(&af); err != nil { return false, interactions, fmt.Errorf("capability frame: %w", err) }
			if af.Kind != "action" { return false, interactions, errors.New("capability emitted non-action frame") }
			if err := chalEnc.Encode(af); err != nil { return false, interactions, fmt.Errorf("forward action: %w", err) }
			interactions++
		case "result":
			var body struct { Passed bool `json:"passed"` }
			if err := json.Unmarshal(cf.Body, &body); err != nil { return false, interactions, errors.New("malformed challenge result") }
			return body.Passed, interactions, nil
		default:
			return false, interactions, fmt.Errorf("unsupported challenge frame kind %q", cf.Kind)
		}
	}
}

func evaluate(r request) response {
	if r.Protocol != protocol { return response{Error: "unsupported protocol"} }
	if !validHash(digest(r.Capability), r.CapabilityHash) { return response{Error: "capability artifact hash mismatch"} }
	if !validHash(digest(r.Challenge), r.ChallengeHash) { return response{Error: "challenge artifact hash mismatch"} }
	if r.WallClockNS == 0 || r.InteractionBudget == 0 { return response{Error: "invalid resource budget"} }
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.WallClockNS)); defer cancel()
	started := time.Now()
	passed, interactions, err := runInteraction(ctx, r)
	elapsed := time.Since(started)
	if err != nil { return response{Error: err.Error(), Interactions: interactions, WallClockNS: uint64(elapsed.Nanoseconds())} }
	if ctx.Err() != nil { return response{Error: "wall-clock budget exceeded", Interactions: interactions, WallClockNS: uint64(elapsed.Nanoseconds())} }
	return response{Passed: passed, Interactions: interactions, WallClockNS: uint64(elapsed.Nanoseconds())}
}

func main() {
	s := bufio.NewScanner(os.Stdin); s.Buffer(make([]byte, 1024), 16<<20)
	enc := json.NewEncoder(os.Stdout)
	for s.Scan() {
		var r request
		if err := json.Unmarshal(s.Bytes(), &r); err != nil { _ = enc.Encode(response{Error: "malformed request"}); continue }
		_ = enc.Encode(evaluate(r))
	}
	if err := s.Err(); err != nil && !errors.Is(err, io.EOF) { os.Exit(2) }
}
