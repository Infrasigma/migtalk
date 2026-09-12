//go:build linux

package main

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

func validateScientificCgroup() error {
    if os.Getenv("ACE_REQUIRE_CGROUP") != "1" {
        return errors.New("scientific execution requires ACE_REQUIRE_CGROUP=1")
    }
    required := os.Getenv("ACE_CGROUP_V2_PATH")
    if required == "" || !strings.HasPrefix(required, "/") || strings.Contains(required, "..") {
        return errors.New("scientific execution requires a safe absolute ACE_CGROUP_V2_PATH")
    }

    data, err := os.ReadFile("/proc/self/cgroup")
    if err != nil {
        return fmt.Errorf("read cgroup membership: %w", err)
    }
    want := "0::" + filepath.Clean(required)
    found := false
    for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
        if strings.TrimSpace(line) == want {
            found = true
            break
        }
    }
    if !found {
        return fmt.Errorf("evaluator is not in required cgroup: got %q want %q", strings.TrimSpace(string(data)), want)
    }

    root := filepath.Join("/sys/fs/cgroup", filepath.Clean(required))
    requiredFiles := []string{"cpu.max", "memory.max", "memory.swap.max", "pids.max"}
    for _, name := range requiredFiles {
        b, err := os.ReadFile(filepath.Join(root, name))
        if err != nil {
            return fmt.Errorf("read %s: %w", name, err)
        }
        value := strings.TrimSpace(string(b))
        if value == "max" || strings.HasPrefix(value, "max ") {
            return fmt.Errorf("scientific cgroup has unlimited %s", name)
        }
    }

    swap, err := os.ReadFile(filepath.Join(root, "memory.swap.max"))
    if err != nil {
        return fmt.Errorf("read memory.swap.max: %w", err)
    }
    if strings.TrimSpace(string(swap)) != "0" {
        return fmt.Errorf("scientific cgroup must disable swap, memory.swap.max=%q", strings.TrimSpace(string(swap)))
    }

    return nil
}
