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
    "syscall"
    "time"
)

const protocol = "ACE-BLIND-2"
const maxArtifactBytes = 16 << 20
const maxChildOutputBytes = 16 << 20

type frame struct {
    Kind string          `json:"kind"`
    Body json.RawMessage `json:"body,omitempty"`
}

type request struct {
    Protocol          string `json:"protocol_version"`
    Plan              string `json:"plan_commitment"`
    Capability        []byte `json:"capability_artifact"`
    CapabilityHash    string `json:"capability_artifact_sha256"`
    Challenge         []byte `json:"challenge_artifact"`
    ChallengeHash     string `json:"challenge_artifact_sha256"`
    WallClockNS       uint64 `json:"wall_clock_ns"`
    InteractionBudget uint64 `json:"interaction_budget"`
}

type response struct {
    Passed        bool   `json:"passed"`
    Interactions  uint64 `json:"interactions"`
    WallClockNS   uint64 `json:"wall_clock_ns"`
    Error         string `json:"error,omitempty"`
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validHash(got, want string) bool { return len(want) == 64 && strings.EqualFold(got, want) }

func writeTempExecutable(b []byte) (string, error) {
    if len(b) == 0 || len(b) > maxArtifactBytes { return "", errors.New("artifact size out of bounds") }
    f, err := os.CreateTemp("", "ace-evaluator-artifact-*")
    if err != nil { return "", err }
    path := f.Name()
    if _, err := f.Write(b); err != nil { _ = f.Close(); _ = os.Remove(path); return "", err }
    if err := f.Close(); err != nil { _ = os.Remove(path); return "", err }
    if err := os.Chmod(path, 0700); err != nil { _ = os.Remove(path); return "", err }
    return path, nil
}

func startIsolated(cmd *exec.Cmd) error {
    if cmd.SysProcAttr == nil {
        cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
    } else {
        cmd.SysProcAttr.Setpgid = true
        cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
    }
    return cmd.Start()
}

func killProcessGroup(cmd *exec.Cmd) {
    if cmd.Process == nil { return }
    _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    _ = cmd.Process.Kill()
}

func runInteraction(ctx context.Context, r request, sandbox bool) (bool, uint64, error) {
    capPath, err := writeTempExecutable(r.Capability)
    if err != nil { return false, 0, err }
    defer os.Remove(capPath)
    chalPath, err := writeTempExecutable(r.Challenge)
    if err != nil { return false, 0, err }
    defer os.Remove(chalPath)

    var capCmd, chalCmd *exec.Cmd
    var capCleanup, chalCleanup func()
    if sandbox {
        capCmd, capCleanup, err = newSandboxedCommand(ctx, r.Capability, capPath)
        if err != nil { return false, 0, fmt.Errorf("prepare capability sandbox: %w", err) }
        chalCmd, chalCleanup, err = newSandboxedCommand(ctx, r.Challenge, chalPath)
        if err != nil { capCleanup(); return false, 0, fmt.Errorf("prepare challenge sandbox: %w", err) }
    } else {
        capCmd = exec.CommandContext(ctx, capPath)
        chalCmd = exec.CommandContext(ctx, chalPath)
        capCleanup = func() {}
        chalCleanup = func() {}
    }

    capIn, err := capCmd.StdinPipe(); if err != nil { capCleanup(); chalCleanup(); return false, 0, err }
    capOut, err := capCmd.StdoutPipe(); if err != nil { capCleanup(); chalCleanup(); return false, 0, err }
    chalIn, err := chalCmd.StdinPipe(); if err != nil { capCleanup(); chalCleanup(); return false, 0, err }
    chalOut, err := chalCmd.StdoutPipe(); if err != nil { capCleanup(); chalCleanup(); return false, 0, err }
    capCmd.Stderr = io.Discard
    chalCmd.Stderr = io.Discard

    if err := startIsolated(capCmd); err != nil { capCleanup(); chalCleanup(); return false, 0, fmt.Errorf("start capability: %w", err) }
    if err := startIsolated(chalCmd); err != nil {
        killProcessGroup(capCmd)
        _ = capCmd.Wait()
        capCleanup()
        chalCleanup()
        return false, 0, fmt.Errorf("start challenge: %w", err)
    }
    defer func() {
        _ = capIn.Close()
        _ = chalIn.Close()
        killProcessGroup(capCmd)
        killProcessGroup(chalCmd)
        _ = capCmd.Wait()
        _ = chalCmd.Wait()
        capCleanup()
        chalCleanup()
    }()

    capDec := json.NewDecoder(io.LimitReader(bufio.NewReader(capOut), maxChildOutputBytes))
    capEnc := json.NewEncoder(capIn)
    chalDec := json.NewDecoder(io.LimitReader(bufio.NewReader(chalOut), maxChildOutputBytes))
    chalEnc := json.NewEncoder(chalIn)
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
    if err := validateScientificCgroup(); err != nil {
        return response{Error: "scientific host boundary rejected: " + err.Error()}
    }
    return evaluateInternal(r, true)
}

// evaluateForTest intentionally bypasses the scientific host/sandbox contract.
// It is only for protocol-unit tests. Scientific execution always enters via evaluate.
func evaluateForTest(r request) response { return evaluateInternal(r, false) }

func evaluateInternal(r request, sandbox bool) response {
    if r.Protocol != protocol { return response{Error: "unsupported protocol"} }
    if !validHash(digest(r.Capability), r.CapabilityHash) { return response{Error: "capability artifact hash mismatch"} }
    if !validHash(digest(r.Challenge), r.ChallengeHash) { return response{Error: "challenge artifact hash mismatch"} }
    if r.WallClockNS == 0 || r.InteractionBudget == 0 { return response{Error: "invalid resource budget"} }
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.WallClockNS))
    defer cancel()
    started := time.Now()
    passed, interactions, err := runInteraction(ctx, r, sandbox)
    elapsed := time.Since(started)
    if err != nil { return response{Error: err.Error(), Interactions: interactions, WallClockNS: uint64(elapsed.Nanoseconds())} }
    if ctx.Err() != nil { return response{Error: "wall-clock budget exceeded", Interactions: interactions, WallClockNS: uint64(elapsed.Nanoseconds())} }
    return response{Passed: passed, Interactions: interactions, WallClockNS: uint64(elapsed.Nanoseconds())}
}

func main() {
    if len(os.Args) == 3 && os.Args[1] == sandboxChildFlag {
        if err := runSandboxChild(os.Args[1:]); err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        return
    }

    s := bufio.NewScanner(os.Stdin)
    s.Buffer(make([]byte, 1024), 16<<20)
    enc := json.NewEncoder(os.Stdout)
    for s.Scan() {
        var r request
        if err := json.Unmarshal(s.Bytes(), &r); err != nil {
            _ = enc.Encode(response{Error: "malformed request"})
            continue
        }
        _ = enc.Encode(evaluate(r))
    }
    if err := s.Err(); err != nil && !errors.Is(err, io.EOF) { os.Exit(2) }
}
