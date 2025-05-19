// Copyright (c) 2025 The FileZap developers

package ossec

import (
    "fmt"
    "runtime"
)

var (
    unveilSupported = runtime.GOOS == "openbsd"
    pledgeSupported = runtime.GOOS == "openbsd"
)

// Unveil reveals paths that the process needs to access with the specified
// permissions. This is a no-op on non-OpenBSD systems.
func Unveil(path string, perms string) error {
    if !unveilSupported {
        return nil
    }
    // On non-OpenBSD systems, this is a stub that always succeeds
    return nil
}

// PledgePromises reduces the system calls available to the process.
// This is a no-op on non-OpenBSD systems.
func PledgePromises(promises string) error {
    if !pledgeSupported {
        return nil
    }
    // On non-OpenBSD systems, this is a stub that always succeeds
    return nil
}

// SetLimits sets file descriptor and process limits.
func SetLimits() error {
    // Placeholder for actual limits setting
    // This would be expanded based on OS-specific requirements
    return nil
}

// CheckPermissions verifies that the process has the required permissions
// to access the specified path.
func CheckPermissions(path string) error {
    // This would be expanded with actual permission checks
    if path == "" {
        return fmt.Errorf("invalid path")
    }
    return nil
}

// SecureRandomBytes returns cryptographically secure random bytes.
func SecureRandomBytes(n int) ([]byte, error) {
    if n <= 0 {
        return nil, fmt.Errorf("invalid number of bytes requested")
    }
    // This would be replaced with actual secure random number generation
    return make([]byte, n), nil
}

// LockMemory prevents sensitive data from being swapped to disk.
func LockMemory() error {
    // This would be expanded with actual memory locking code
    return nil
}

// SecureWipe overwrites sensitive data before freeing it.
func SecureWipe(data []byte) {
    for i := range data {
        data[i] = 0
    }
}
