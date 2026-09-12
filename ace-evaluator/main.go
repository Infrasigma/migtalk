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
	Plan     string `json:"plan_commitment"`
	Artifact []byte `json:"capability_artifact"`
	Hash     string `json:"capability_artifact_sha256"`
	Challenge []byte `json:"challenge_artifact"`
	ChallengeHash string `json:"challenge_artifact_sha256"`
	Budget budget `json:"budget"`
}

type budget struct { WallClockNS int64 `json:"wall_clock_ns"`; Interactions int64 `json:"interactions"` }

type result struct { Passed bool `json:"passed"`; Error string `json:"error,omitempty"`; Usage map[string]int64 `json:"usage,omitempty"` }

func digest(b []byte) string { h:=sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validHash(got, want string) bool { return strings.EqualFold(got,want) && len(want)==64 }

func runArtifact(ctx context.Context, artifact []byte, expected string, input []byte) ([]byte,error) {
	if !validHash(digest(artifact),expected) { return nil,errors.New("artifact hash mismatch") }
	f,err:=os.CreateTemp("","ace-evaluator-artifact-*"); if err!=nil{return nil,err}; path:=f.Name(); defer os.Remove(path)
	if _,err=f.Write(artifact); err!=nil { f.Close(); return nil,err }; if err=f.Close(); err!=nil{return nil,err}
	if err=os.Chmod(path,0700); err!=nil{return nil,err}
	cmd:=exec.CommandContext(ctx,path); cmd.Stdin=strings.NewReader(string(input)); out,err:=cmd.Output(); if ctx.Err()!=nil{return nil,ctx.Err()}; if err!=nil{return nil,fmt.Errorf("artifact execution failed: %w",err)}
	return out,nil
}

func evaluate(r request) result {
	if r.Protocol!=protocol{return result{Error:"unsupported protocol"}}
	if !validHash(digest(r.Artifact),r.Hash){return result{Error:"capability artifact hash mismatch"}}
	if !validHash(digest(r.Challenge),r.ChallengeHash){return result{Error:"challenge artifact hash mismatch"}}
	if r.Budget.WallClockNS<=0 || r.Budget.Interactions<=0{return result{Error:"invalid resource budget"}}
	ctx,cancel:=context.WithTimeout(context.Background(),time.Duration(r.Budget.WallClockNS)); defer cancel()
	// The evaluator never interprets challenge semantics. It only launches the
	// sealed challenge executable and passes opaque capability bytes to it.
	challengeInput:=r.Artifact
	out,err:=runArtifact(ctx,r.Challenge,r.ChallengeHash,challengeInput); if err!=nil{return result{Error:err.Error()}}
	// Challenge output is an evaluator-neutral JSON result. No per-challenge
	// information is returned to the learner harness beyond the final boolean.
	var cr struct{Passed bool `json:"passed"`}; if err:=json.Unmarshal(out,&cr);err!=nil{return result{Error:"malformed challenge result"}}
	return result{Passed:cr.Passed,Usage:map[string]int64{"interactions":1,"wall_clock_ns":time.Since(time.Now()).Nanoseconds()}}
}

func main(){
	s:=bufio.NewScanner(os.Stdin); s.Buffer(make([]byte,1024),4<<20)
	enc:=json.NewEncoder(os.Stdout)
	for s.Scan(){
		var r request; if err:=json.Unmarshal(s.Bytes(),&r);err!=nil { enc.Encode(result{Error:"malformed request"}); continue }
		if len(r.Artifact)==0||len(r.Challenge)==0 {enc.Encode(result{Error:"missing artifact"});continue}
		enc.Encode(evaluate(r))
	}
	if err:=s.Err();err!=nil && !errors.Is(err,io.EOF){os.Exit(2)}
}
