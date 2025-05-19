// Copyright (c) 2025 The FileZap developers

package limits

import (
    "fmt"
    "runtime"
    "syscall"
)

const (
    // DefaultMaxFileDescriptors is the default maximum number of file descriptors.
    DefaultMaxFileDescriptors = 16384

    // DefaultMaxMemory is the default maximum memory in bytes.
    DefaultMaxMemory = 4 * 1024 * 1024 * 1024 // 4GB
)

// SetLimits sets process limits for better performance and stability.
// This includes setting maximum file descriptors and virtual memory limits.
func SetLimits() error {
    if err := setFileDescriptorLimit(); err != nil {
        return fmt.Errorf("failed to set file descriptor limit: %v", err)
    }

    if err := setMemoryLimit(); err != nil {
        return fmt.Errorf("failed to set memory limit: %v", err)
    }

    return nil
}

// setFileDescriptorLimit tries to raise the process's file descriptor limit
// to the maximum allowed.
func setFileDescriptorLimit() error {
    if runtime.GOOS == "windows" {
        // Windows doesn't need explicit file descriptor limits
        return nil
    }

    var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        return err
    }

    rLimit.Cur = DefaultMaxFileDescriptors
    if rLimit.Max < DefaultMaxFileDescriptors {
        rLimit.Cur = rLimit.Max
    }

    return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
}

// setMemoryLimit sets the maximum virtual memory limit for the process.
func setMemoryLimit() error {
    if runtime.GOOS == "windows" {
        // Windows handles memory limits differently
        return nil
    }

    var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_AS, &rLimit)
    if err != nil {
        return err
    }

    rLimit.Cur = DefaultMaxMemory
    if rLimit.Max < DefaultMaxMemory {
        rLimit.Cur = rLimit.Max
    }

    return syscall.Setrlimit(syscall.RLIMIT_AS, &rLimit)
}

// GetFileDescriptorLimit returns the current and maximum file descriptor limits.
func GetFileDescriptorLimit() (uint64, uint64, error) {
    if runtime.GOOS == "windows" {
        return 0, 0, fmt.Errorf("not supported on Windows")
    }

    var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        return 0, 0, err
    }

    return rLimit.Cur, rLimit.Max, nil
}

// GetMemoryLimit returns the current and maximum virtual memory limits.
func GetMemoryLimit() (uint64, uint64, error) {
    if runtime.GOOS == "windows" {
        return 0, 0, fmt.Errorf("not supported on Windows")
    }

    var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_AS, &rLimit)
    if err != nil {
        return 0, 0, err
    }

    return rLimit.Cur, rLimit.Max, nil
}

// SetMaxThreads sets the maximum number of threads for the process.
func SetMaxThreads(max uint64) error {
    if runtime.GOOS == "windows" {
        return nil
    }

    var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_NPROC, &rLimit)
    if err != nil {
        return err
    }

    rLimit.Cur = max
    if rLimit.Max < max {
        rLimit.Cur = rLimit.Max
    }

    return syscall.Setrlimit(syscall.RLIMIT_NPROC, &rLimit)
}
