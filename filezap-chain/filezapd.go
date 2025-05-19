// Copyright (c) 2025 The FileZap developers

package main

import (
    "fmt"
    "net/http"
    _ "net/http/pprof"
    "os"
    "path/filepath"
    "runtime"
    "runtime/debug"
    "runtime/pprof"
    "runtime/trace"

    "github.com/VetheonGames/FileZap/filezap-chain/database"
    "github.com/VetheonGames/FileZap/filezap-chain/limits"
    "github.com/VetheonGames/FileZap/filezap-chain/types"

    fzcfg "github.com/VetheonGames/FileZap/filezap-chain/config"
    fzsrv "github.com/VetheonGames/FileZap/filezap-chain/server"
)

const (
    blockDbNamePrefix  = "filezap"
    contentDbNamePrefix = "content"
)

var (
    appConfig  *fzcfg.Config
    fileZapLog *types.Logger
)

// filezapdMain is the real main function for filezapd.
func filezapdMain() error {
    var err error
    appConfig, _, err = fzcfg.LoadConfig()
    if err != nil {
        return err
    }

    // Initialize logging
    setupLogging()

    // Get a channel that will be closed when a shutdown signal has been triggered
    quit := interruptSignal()
    defer fileZapLog.Info("Shutdown complete")

    // Show version at startup
    fileZapLog.Infof("FileZap version %s", versionString())

    // Enable http profiling server if requested.
    if appConfig.Profile != "" {
        go func() {
            profileRedirect := http.RedirectHandler("/debug/pprof", http.StatusSeeOther)
            http.Handle("/", profileRedirect)
            fileZapLog.Errorf("%v", http.ListenAndServe(appConfig.Profile, nil))
        }()
    }

    // Handle profiling and tracing if requested
    if err := setupProfiling(); err != nil {
        return err
    }

    // Load the block database.
    db, err := loadBlockDB()
    if err != nil {
        fileZapLog.Errorf("%v", err)
        return err
    }
    defer func() {
        fileZapLog.Infof("Gracefully shutting down the database...")
        db.Close()
    }()

    // Load the content hash database
    contentDb, err := loadContentDB()
    if err != nil {
        fileZapLog.Errorf("%v", err)
        return err
    }
    defer func() {
        fileZapLog.Infof("Gracefully shutting down the content database...")
        contentDb.Close()
    }()

    // Create and start the server
    s, err := fzsrv.New(appConfig.ActiveNet, appConfig.Listen, db, contentDb)
    if err != nil {
        fileZapLog.Errorf("Unable to start server: %v", err)
        return err
    }

    s.Start()
    defer func() {
        fileZapLog.Info("Gracefully shutting down the server...")
        s.Stop()
        s.WaitForShutdown()
    }()

    // Wait for shutdown signal
    <-quit
    return nil
}

func setupProfiling() error {
    if appConfig.CPUProfile != "" {
        f, err := os.Create(appConfig.CPUProfile)
        if err != nil {
            fileZapLog.Errorf("Unable to create CPU profile: %v", err)
            return err
        }
        pprof.StartCPUProfile(f)
        defer f.Close()
        defer pprof.StopCPUProfile()
    }

    if appConfig.MemoryProfile != "" {
        f, err := os.Create(appConfig.MemoryProfile)
        if err != nil {
            fileZapLog.Errorf("Unable to create memory profile: %v", err)
            return err
        }
        defer f.Close()
        defer pprof.WriteHeapProfile(f)
        defer runtime.GC()
    }

    if appConfig.TraceProfile != "" {
        f, err := os.Create(appConfig.TraceProfile)
        if err != nil {
            fileZapLog.Errorf("Unable to create execution trace: %v", err)
            return err
        }
        trace.Start(f)
        defer f.Close()
        defer trace.Stop()
    }

    return nil
}

func loadContentDB() (database.DB, error) {
    if appConfig.DbType == "memdb" {
        fileZapLog.Infof("Creating content database in memory.")
        db, err := database.Create(appConfig.DbType, "", appConfig.ActiveNet.Net)
        if err != nil {
            return nil, err
        }
        return db, nil
    }

    dbPath := contentDbPath(appConfig.DbType)
    fileZapLog.Infof("Loading content database from '%s'", dbPath)

    db, err := database.Open(appConfig.DbType, dbPath, appConfig.ActiveNet.Net)
    if err != nil {
        if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode != database.ErrDbDoesNotExist {
            return nil, err
        }

        err = os.MkdirAll(appConfig.DataDir, 0700)
        if err != nil {
            return nil, err
        }
        db, err = database.Create(appConfig.DbType, dbPath, appConfig.ActiveNet.Net)
        if err != nil {
            return nil, err
        }
    }

    fileZapLog.Info("Content database loaded")
    return db, nil
}

func loadBlockDB() (database.DB, error) {
    if appConfig.DbType == "memdb" {
        fileZapLog.Infof("Creating block database in memory.")
        db, err := database.Create(appConfig.DbType, "", appConfig.ActiveNet.Net)
        if err != nil {
            return nil, err
        }
        return db, nil
    }

    dbPath := blockDbPath(appConfig.DbType)
    fileZapLog.Infof("Loading block database from '%s'", dbPath)

    db, err := database.Open(appConfig.DbType, dbPath, appConfig.ActiveNet.Net)
    if err != nil {
        if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode != database.ErrDbDoesNotExist {
            return nil, err
        }

        err = os.MkdirAll(appConfig.DataDir, 0700)
        if err != nil {
            return nil, err
        }
        db, err = database.Create(appConfig.DbType, dbPath, appConfig.ActiveNet.Net)
        if err != nil {
            return nil, err
        }
    }

    fileZapLog.Info("Block database loaded")
    return db, nil
}

func contentDbPath(dbType string) string {
    dbName := contentDbNamePrefix + "_" + dbType
    if dbType == "sqlite" {
        dbName = dbName + ".db"
    }
    return filepath.Join(appConfig.DataDir, dbName)
}

func blockDbPath(dbType string) string {
    dbName := blockDbNamePrefix + "_" + dbType
    if dbType == "sqlite" {
        dbName = dbName + ".db"
    }
    return filepath.Join(appConfig.DataDir, dbName)
}

func setupLogging() {
    backend := types.NewLogBackend()
    fileZapLog = types.NewSubLogger("FZPD", backend)
    fzsrv.UseLogger(types.NewSubLogger("SRVR", backend))
}

func main() {
    if os.Getenv("GOGC") == "" {
        debug.SetGCPercent(10)
    }

    if err := limits.SetLimits(); err != nil {
        fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
        os.Exit(1)
    }

    if err := filezapdMain(); err != nil {
        os.Exit(1)
    }
}
