package vpn

import (
    "github.com/songgao/water"
)

// platformConfig configures platform-specific TUN interface settings
// Implemented differently for each OS
func platformConfig(cfg *water.Config, name string)

// setupInterface configures the TUN interface with IP settings
// Implemented differently for each OS
func setupInterface(iface *water.Interface, config TUNConfig) error
