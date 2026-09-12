//go:build linux

package main

import (
    "os"
    "strings"
    "testing"
)

func TestValidateScientificCgroupFailsClosedWithoutContract(t *testing.T) {
    t.Setenv("ACE_REQUIRE_CGROUP", "")
    t.Setenv("ACE_CGROUP_V2_PATH", "")
    if err := validateScientificCgroup(); err == nil {
        t.Fatal("expected missing scientific cgroup contract to fail closed")
    }
}

func TestValidateScientificCgroupRejectsUnsafePath(t *testing.T) {
    t.Setenv("ACE_REQUIRE_CGROUP", "1")
    t.Setenv("ACE_CGROUP_V2_PATH", "../escape")
    err := validateScientificCgroup()
    if err == nil || !strings.Contains(err.Error(), "safe absolute") {
        t.Fatalf("expected unsafe cgroup path rejection, got %v", err)
    }
}

func TestTempExecutableNameDoesNotContainTargetHash(t *testing.T) {
    target := strings.Repeat("ab", 32)
    p, err := writeTempExecutable([]byte("not-an-executable"))
    if err != nil {
        // The helper only validates size; malformed ELF is irrelevant here.
        t.Fatal(err)
    }
    defer os.Remove(p)
    if strings.Contains(p, target) {
        t.Fatalf("temporary path leaked target hash: %s", p)
    }
    if strings.Contains(strings.ToLower(p), "challenge") {
        t.Fatalf("temporary path contains target-identifying challenge marker: %s", p)
    }
}
