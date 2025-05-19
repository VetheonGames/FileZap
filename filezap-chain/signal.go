// Copyright (c) 2025 The FileZap developers

package main

import (
    "os"
    "os/signal"
    "syscall"
)

// interruptListener returns a channel that will receive a signal when an interrupt
// occurs.
func interruptSignal() <-chan struct{} {
    c := make(chan struct{})
    interruptChan := make(chan os.Signal, 1)
    signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-interruptChan
        signal.Stop(interruptChan)
        close(c)
    }()
    return c
}
