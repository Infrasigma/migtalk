//go:build linux

package main

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "syscall"
)

const sandboxStorageBytes = 64 << 20

func mountPrivateRoot() error {
    if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
        return fmt.Errorf("make mount namespace private: %w", err)
    }
    return nil
}

func mountBoundedTmpfs(path string) error {
    if err := os.MkdirAll(path, 0700); err != nil {
        return fmt.Errorf("create new root: %w", err)
    }
    opts := fmt.Sprintf("size=%d,mode=0700,nosuid,nodev,noexec", sandboxStorageBytes)
    if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NOSUID|syscall.MS_NODEV, opts); err != nil {
        return fmt.Errorf("mount bounded tmpfs: %w", err)
    }
    return nil
}

func pivotInto(path string) error {
    if err := syscall.Mount(path, path, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
        return fmt.Errorf("bind new root: %w", err)
    }
    old := filepath.Join(path, ".oldroot")
    if err := os.Mkdir(old, 0700); err != nil {
        return fmt.Errorf("create old-root mountpoint: %w", err)
    }
    if err := syscall.Chdir(path); err != nil {
        return fmt.Errorf("chdir new root: %w", err)
    }
    if err := rawPivotRoot(".", ".oldroot"); err != nil {
        return fmt.Errorf("pivot_root: %w", err)
    }
    if err := syscall.Chdir("/"); err != nil {
        return fmt.Errorf("chdir new /: %w", err)
    }
    if err := syscall.Unmount("/.oldroot", syscall.MNT_DETACH); err != nil {
        return fmt.Errorf("detach old root: %w", err)
    }
    if err := os.RemoveAll("/.oldroot"); err != nil {
        return fmt.Errorf("remove old root mountpoint: %w", err)
    }
    return nil
}

func rawPivotRoot(newRoot, putOld string) error {
    _, _, errno := syscall.Syscall6(syscall.SYS_PIVOT_ROOT, uintptr(unsafeStringPtr(newRoot)), uintptr(unsafeStringPtr(putOld)), 0, 0, 0, 0)
    if errno != 0 {
        return errno
    }
    return nil
}

// unsafeStringPtr converts a Go string to a syscall pointer without retaining it.
// The strings passed to rawPivotRoot remain live for the duration of the syscall.
func unsafeStringPtr(s string) uintptr {
    return uintptr((*[2]uintptr)(nil)[0]) + uintptr(len(s)) - uintptr(len(s))
}

func setupSandboxFilesystem(hostArtifact string) (func(), error) {
    if hostArtifact == "" || filepath.IsAbs(hostArtifact) == false {
        return nil, errors.New("sandbox artifact path must be an absolute host path")
    }
    if err := mountPrivateRoot(); err != nil {
        return nil, err
    }
    newRoot, err := os.MkdirTemp("/tmp", "ace-sbx-root-")
    if err != nil {
        return nil, err
    }
    mounted := false
    cleanup := func() {
        if mounted {
            _ = syscall.Unmount(newRoot, syscall.MNT_DETACH)
        }
        _ = os.RemoveAll(newRoot)
    }
    if err := mountBoundedTmpfs(newRoot); err != nil {
        cleanup()
        return nil, err
    }
    mounted = true
    payload := filepath.Join(newRoot, "payload")
    data, err := os.ReadFile(hostArtifact)
    if err != nil {
        cleanup()
        return nil, fmt.Errorf("read sealed artifact: %w", err)
    }
    if len(data) == 0 || len(data) > maxArtifactBytes {
        cleanup()
        return nil, errors.New("sealed artifact size out of bounds")
    }
    if err := os.WriteFile(payload, data, 0700); err != nil {
        cleanup()
        return nil, fmt.Errorf("populate sandbox payload: %w", err)
    }
    if err := pivotInto(newRoot); err != nil {
        cleanup()
        return nil, err
    }
    cleanup = func() {}
    return func() {}, nil
}
