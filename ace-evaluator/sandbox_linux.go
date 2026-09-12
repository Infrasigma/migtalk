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
	"path/filepath"
	"syscall"
)

func validateStaticELF(b []byte) error {
	f, err := elf.NewFile(bytes.NewReader(b))
	if err != nil { return errors.New("capability/challenge must be a valid ELF executable") }
	for _, ph := range f.Progs {
		if ph.Type == elf.PT_INTERP { return errors.New("dynamically linked ELF is not admitted; executable must be self-contained") }
	}
	return nil
}

func newSandboxRoot(b []byte) (string, error) {
	if err := validateStaticELF(b); err != nil { return "", err }
	root, err := os.MkdirTemp("", "ace-sandbox-root-*")
	if err != nil { return "", err }
	payload := filepath.Join(root, "payload")
	if err := os.WriteFile(payload, b, 0700); err != nil { _ = os.RemoveAll(root); return "", err }
	return root, nil
}

func newSandboxedCommand(ctx context.Context, artifact []byte, _ string) (*exec.Cmd, func(), error) {
	root, err := newSandboxRoot(artifact)
	if err != nil { return nil, nil, err }
	cleanup := func() { _ = os.RemoveAll(root) }

	cmd := exec.CommandContext(ctx, "/payload")
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pdeathsig: syscall.SIGKILL,
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID,
		UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
		GidMappingsEnableSetgroups: false,
		Chroot: root,
	}
	cmd.Env = []string{"PATH=/bin", "HOME=/", "LANG=C"}
	return cmd, cleanup, nil
}

func sandboxSupportCheck() error {
	if _, err := exec.LookPath("/proc/self/exe"); err == nil {
		return nil
	}
	if _, err := os.Stat("/proc"); err != nil { return fmt.Errorf("/proc unavailable: %w", err) }
	return nil
}
