// Copyright (c) 2025 The FileZap developers

package main

import "fmt"

const (
    // semanticVersion identifies the version of FileZap
    semanticVersion = "0.1.0"

    // appMajor is application major version
    appMajor = 0

    // appMinor is application minor version
    appMinor = 1

    // appPatch is application patch version
    appPatch = 0
)

// version returns the version string
func versionString() string {
    return semanticVersion
}

// appVersion returns the version as major.minor.patch
func appVersion() string {
    return fmt.Sprintf("%d.%d.%d", appMajor, appMinor, appPatch)
}
