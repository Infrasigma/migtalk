//go:build linux

package main

import (
    "context"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "testing"
    "time"
)

const sandboxProbeSource = `package main
import (
  "fmt"
  "os"
)
func main() {
  cwd, _ := os.Getwd()
  if cwd != "/" { fmt.Println("bad-cwd"); os.Exit(10) }
  if _, err := os.Stat("/etc/passwd"); err == nil { fmt.Println("host-fs-visible"); os.Exit(11) }
  if _, err := os.Stat("/tmp"); err == nil { fmt.Println("host-tmp-visible"); os.Exit(12) }
  if _, err := os.Stat("/dev/shm"); err == nil { fmt.Println("host-shm-visible"); os.Exit(13) }
  if _, err := os.Stat("/proc/1"); err == nil { fmt.Println("host-proc-visible"); os.Exit(14) }
  if os.Getenv("ACE_CHALLENGE_HASH") != "" { fmt.Println("hash-env-visible"); os.Exit(15) }
  if err := os.WriteFile("/marker", []byte("sandbox"), 0600); err != nil { fmt.Println("marker-write-failed"); os.Exit(16) }

  // The production sandbox must enforce a finite inode ceiling. We expect
  // creation to fail before the host can be exhausted. A successful creation
  // of every file is a containment failure, not a scientific result.
  for i := 0; i < 2048; i++ {
    name := fmt.Sprintf("/f-%04d", i)
    if err := os.WriteFile(name, []byte{}, 0600); err != nil {
      if i < 100 { fmt.Println("inode-limit-too-small"); os.Exit(17) }
      return
    }
  }
  fmt.Println("inode-limit-not-enforced")
  os.Exit(18)
}`

func buildSandboxProbe(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    src := filepath.Join(dir, "main.go")
    out := filepath.Join(dir, "probe")
    if err := os.WriteFile(src, []byte(sandboxProbeSource), 0600); err != nil { t.Fatal(err) }
    cmd := exec.Command("go", "build", "-o", out, src)
    cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
    if output, err := cmd.CombinedOutput(); err != nil { t.Fatalf("build sandbox probe: %v\n%s", err, output) }
    return out
}

func TestSandboxAdversarialFilesystemAndBlindness(t *testing.T) {
    if runtime.GOOS != "linux" { t.Skip("Linux namespaces required") }
    if os.Getenv("ACE_RUN_SANDBOX_TESTS") != "1" { t.Skip("set ACE_RUN_SANDBOX_TESTS=1 on an admitted Linux sandbox runner") }

    probe := buildSandboxProbe(t)
    artifact, err := os.ReadFile(probe)
    if err != nil { t.Fatal(err) }

    old := os.Args[0]
    os.Args[0] = os.Getenv("ACE_SANDBOX_EXECUTABLE")
    if os.Args[0] == "" { t.Fatal("ACE_SANDBOX_EXECUTABLE must point to the built evaluator") }
    defer func() { os.Args[0] = old }()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    cmd, cleanup, err := newSandboxedCommand(ctx, artifact, probe)
    if err != nil { t.Fatalf("sandbox construction: %v", err) }
    defer cleanup()
    if err := cmd.Run(); err != nil { t.Fatalf("sandbox probe escaped or failed: %v", err) }
}
