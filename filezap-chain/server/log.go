// Copyright (c) 2025 The FileZap developers

package server

import (
    "github.com/VetheonGames/FileZap/filezap-chain/types"
)

var log *types.Logger

// UseLogger sets the package-level logger.
func UseLogger(logger *types.Logger) {
    log = logger
}

func init() {
    // Create default logger
    backend := types.NewLogBackend()
    log = types.NewSubLogger("SRVR", backend)
}
