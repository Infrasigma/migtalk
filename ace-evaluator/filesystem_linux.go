//go:build linux

package main

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "syscall"
    "unsafe"
)

const sandboxStorageBytes = 16 << 20
const sandboxStorageInodes = 1024

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
    opts := fmt.Sprintf("size=%d,nr_inodes=%d,mode=0700,nosuid,nodev", sandboxStorageBytes, sandboxStorageInodes)
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
        return fmt.Errorf("chdir new root: %w", err)
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
    newBuf := append([]byte(newRoot), 0)
    oldBuf := append([]byte(putOld), 0)
    _, _, errno := syscall.Syscall6(
        syscall.SYS_PIVOT_ROOT,
        uintptr(unsafe.Pointer(&newBuf[0])),
        uintptr(unsafe.Pointer(&oldBuf[0])),
        0, 0, 0, 0,
    )
    if errno != 0 {
        return errno
    }
    return nil
}

// setupSandboxFilesystem runs after the child enters its private user and
// mount namespaces. The new root is a bounded tmpfs containing only payload.
func setupSandboxFilesystem(hostArtifact string) error {
    if hostArtifact == "" || !filepath.IsAbs(hostArtifact) {
        return errors.New("sandbox artifact path must be an absolute host path")
    }
    if err := mountPrivateRoot(); err != nil {
        return err
    }
    newRoot, err := os.MkdirTemp("/tmp", "ace-sbx-root-")
    if err != nil {
        return err
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
        return err
    }
    mounted = true

    data, err := os.ReadFile(hostArtifact)
    if err != nil {
        cleanup()
        return fmt.Errorf("read sealed artifact: %w", err)
    }
    if len(data) == 0 || len(data) > maxArtifactBytes {
        cleanup()
        return errors.New("sealed artifact size out of bounds")
    }
    payload := filepath.Join(newRoot, "payload")
    if err := os.WriteFile(payload, data, 0700); err != nil {
        cleanup()
        return fmt.Errorf("populate sandbox payload: %w", err)
    }
    if err := pivotInto(newRoot); err != nil {
        cleanup()
        return err
    }
    return nil
}
