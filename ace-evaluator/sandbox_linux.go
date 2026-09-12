//go:build linux

package main

import (
    "bytes"
    "context"
    "debug/elf"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "syscall"
)

const sandboxChildFlag = "--ace-sandbox-child"

func validateStaticELF(b []byte) error {
    f, err := elf.NewFile(bytes.NewReader(b))
    if err != nil {
        return errors.New("capability/challenge must be a valid ELF executable")
    }
    for _, ph := range f.Progs {
        if ph.Type == elf.PT_INTERP {
            return errors.New("dynamically linked ELF is not admitted; executable must be self-contained")
        }
    }
    return nil
}

func newSandboxedCommand(ctx context.Context, artifact []byte, hostArtifact string) (*exec.Cmd, func(), error) {
    if err := validateStaticELF(artifact); err != nil {
        return nil, nil, err
    }
    if hostArtifact == "" {
        return nil, nil, errors.New("sandbox requires a sealed host artifact path")
    }

    cmd := exec.CommandContext(ctx, os.Args[0], sandboxChildFlag, hostArtifact)
    cmd.Dir = "/"
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
        Pdeathsig: syscall.SIGKILL,
        Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID,
        UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
        GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
        GidMappingsEnableSetgroups: false,
    }
    // The sandbox child receives only an opaque random temporary artifact path.
    // No challenge/family/artifact hash is included in argv, env, or filenames.
    cmd.Env = []string{
        "PATH=/usr/bin:/bin",
        "HOME=/",
        "LANG=C",
        "LC_ALL=C",
        "ACE_SANDBOX_CHILD=1",
    }
    return cmd, func() {}, nil
}

func runSandboxChild(args []string) error {
    if len(args) != 2 || args[0] != sandboxChildFlag {
        return errors.New("invalid sandbox child invocation")
    }
    if err := setupSandboxFilesystem(args[1]); err != nil {
        return err
    }
    if err := syscall.Chdir("/"); err != nil {
        return fmt.Errorf("sandbox chdir: %w", err)
    }
    if err := syscall.Exec("/payload", []string{"/payload"}, []string{"PATH=/usr/bin:/bin", "HOME=/", "LANG=C", "LC_ALL=C"}); err != nil {
        return fmt.Errorf("exec sealed payload: %w", err)
    }
    return nil
}

func sandboxSupportCheck() error {
    if os.Getuid() < 0 {
        return errors.New("invalid invoking UID")
    }
    if _, err := os.Stat("/proc/self/ns"); err != nil {
        return fmt.Errorf("Linux namespace support unavailable: %w", err)
    }
    return nil
}
