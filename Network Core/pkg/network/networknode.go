package network

import (
    "fmt"
)

// Close shuts down the network node and releases resources
func (n *NetworkNode) Close() error {
    var errs []error

    if err := n.dht.Close(); err != nil {
        errs = append(errs, fmt.Errorf("failed to close DHT: %w", err))
    }

    if err := n.host.Close(); err != nil {
        errs = append(errs, fmt.Errorf("failed to close host: %w", err))
    }

    if len(errs) > 0 {
        return fmt.Errorf("errors closing network node: %v", errs)
    }
    
    return nil
}
