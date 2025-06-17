package ui

import (
    "fmt"
    "strconv"
    "time"
    
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/storage"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
    
    "github.com/VetheonGames/FileZap/Client/pkg/client"
)

type FileZapUI struct {
    app        fyne.App
    mainWindow fyne.Window
    client     *client.FileZapClient
    config     *client.ClientConfig

    // UI Components
    peerList     *widget.List
    peerData     []string
    selectedPeer int
    status       *widget.Label
    storageStats *widget.Label
}

func NewFileZapUI() *FileZapUI {
    ui := &FileZapUI{
        app:          app.New(),
        peerData:     make([]string, 0),
        selectedPeer: -1,
    }

    // Create default config
    ui.config = client.DefaultClientConfig()

    // Initialize client
    client, err := client.NewFileZapClient(ui.config)
    if err != nil {
        panic(fmt.Sprintf("Failed to create client: %v", err))
    }
    ui.client = client

    ui.mainWindow = ui.app.NewWindow("FileZap")
    ui.setupUI()

    return ui
}

func (ui *FileZapUI) setupUI() {
    // Create tabs for different operations
    tabs := container.NewAppTabs(
        container.NewTabItem("Files", ui.createFilesTab()),
        container.NewTabItem("Network", ui.createNetworkTab()),
        container.NewTabItem("Storage", ui.createStorageTab()),
        container.NewTabItem("Settings", ui.createSettingsTab()),
    )
    tabs.SetTabLocation(container.TabLocationTop)

    // Create status bar
    ui.status = widget.NewLabel("Ready")
    statusBar := container.NewHBox(ui.status)

    // Layout
    content := container.NewBorder(
        nil,
        statusBar,
        nil,
        nil,
        tabs,
    )

    ui.mainWindow.SetContent(content)
}

func (ui *FileZapUI) createFilesTab() fyne.CanvasObject {
    // File upload section
    uploadGroup := widget.NewCard("Upload File", "", container.NewVBox(
        ui.createUploadControls(),
    ))

    // File download section
    downloadGroup := widget.NewCard("Download File", "", container.NewVBox(
        ui.createDownloadControls(),
    ))

    // Report malicious file section
    reportGroup := widget.NewCard("Report File", "", container.NewVBox(
        ui.createReportControls(),
    ))

    return container.NewVBox(
        uploadGroup,
        widget.NewSeparator(),
        downloadGroup,
        widget.NewSeparator(),
        reportGroup,
    )
}

func (ui *FileZapUI) createUploadControls() fyne.CanvasObject {
    inputPath := widget.NewEntry()
    inputPath.SetPlaceHolder("Select file to upload")

    chunkSize := widget.NewEntry()
    chunkSize.SetText("1048576") // Default 1MB chunks

    inputSelect := widget.NewButton("Browse", func() {
        fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
            if err != nil || reader == nil {
                return
            }
            inputPath.SetText(reader.URI().Path())
        }, ui.mainWindow)
        fd.Show()
    })

    uploadButton := widget.NewButtonWithIcon("Upload", theme.UploadIcon(), func() {
        if inputPath.Text == "" {
            dialog.ShowError(fmt.Errorf("please select a file"), ui.mainWindow)
            return
        }
        
        go func() {
            ui.status.SetText("Uploading file...")
            if err := ui.client.UploadFile(inputPath.Text); err != nil {
                dialog.ShowError(err, ui.mainWindow)
                ui.status.SetText("Upload failed")
                return
            }
            ui.status.SetText("Upload complete")
        }()
    })

    return container.NewVBox(
        container.NewBorder(nil, nil, nil, inputSelect, inputPath),
        container.NewGridWithColumns(2,
            widget.NewLabel("Chunk Size (bytes):"),
            chunkSize,
        ),
        uploadButton,
    )
}

func (ui *FileZapUI) createDownloadControls() fyne.CanvasObject {
    zapPath := widget.NewEntry()
    zapPath.SetPlaceHolder("Select .zap file")

    outputPath := widget.NewEntry()
    outputPath.SetPlaceHolder("Select output directory")

    zapSelect := widget.NewButton("Browse", func() {
        fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
            if err != nil || reader == nil {
                return
            }
            zapPath.SetText(reader.URI().Path())
        }, ui.mainWindow)
        fd.SetFilter(storage.NewExtensionFileFilter([]string{".zap"}))
        fd.Show()
    })

    outputSelect := widget.NewButton("Browse", func() {
        fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
            if err != nil || uri == nil {
                return
            }
            outputPath.SetText(uri.Path())
        }, ui.mainWindow)
        fd.Show()
    })

    downloadButton := widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
        if zapPath.Text == "" || outputPath.Text == "" {
            dialog.ShowError(fmt.Errorf("please select both .zap file and output directory"), ui.mainWindow)
            return
        }

        go func() {
            ui.status.SetText("Downloading file...")
            if err := ui.client.DownloadFile(zapPath.Text, outputPath.Text); err != nil {
                dialog.ShowError(err, ui.mainWindow)
                ui.status.SetText("Download failed")
                return
            }
            ui.status.SetText("Download complete")
        }()
    })

    return container.NewVBox(
        container.NewBorder(nil, nil, nil, zapSelect, zapPath),
        container.NewBorder(nil, nil, nil, outputSelect, outputPath),
        downloadButton,
    )
}

func (ui *FileZapUI) createReportControls() fyne.CanvasObject {
    fileID := widget.NewEntry()
    fileID.SetPlaceHolder("File ID to report")

    reason := widget.NewMultiLineEntry()
    reason.SetPlaceHolder("Reason for report")

    reportButton := widget.NewButtonWithIcon("Report File", theme.WarningIcon(), func() {
        if fileID.Text == "" || reason.Text == "" {
            dialog.ShowError(fmt.Errorf("please provide file ID and reason"), ui.mainWindow)
            return
        }

        if err := ui.client.ReportBadFile(fileID.Text, reason.Text); err != nil {
            dialog.ShowError(err, ui.mainWindow)
            return
        }
        dialog.ShowInformation("Report Submitted", "The file has been reported for review", ui.mainWindow)
    })

    return container.NewVBox(
        fileID,
        reason,
        reportButton,
    )
}

func (ui *FileZapUI) createNetworkTab() fyne.CanvasObject {
    // Create peer list
    ui.peerList = widget.NewList(
        func() int { return len(ui.peerData) },
        func() fyne.CanvasObject { 
            return container.NewHBox(
                widget.NewLabel("Template Peer"),
                widget.NewLabel("Status"),
            )
        },
        func(id widget.ListItemID, obj fyne.CanvasObject) {
            peer := ui.peerData[id]
            box := obj.(*fyne.Container)
            box.Objects[0].(*widget.Label).SetText(peer)
            // TODO: Add real peer status
            box.Objects[1].(*widget.Label).SetText("Connected")
        },
    )
    
    // Set OnSelected callback for peer list
    ui.peerList.OnSelected = func(id widget.ListItemID) {
        ui.selectedPeer = int(id)
    }

    reportPeer := widget.NewButton("Report Malicious Peer", func() {
        if ui.selectedPeer < 0 || ui.selectedPeer >= len(ui.peerData) {
            dialog.ShowError(fmt.Errorf("please select a peer to report"), ui.mainWindow)
            return
        }

        reason := widget.NewMultiLineEntry()
        reason.SetPlaceHolder("Enter reason for reporting peer")

        reportDialog := dialog.NewForm(
            "Report Peer",
            "Submit",
            "Cancel",
            []*widget.FormItem{
                widget.NewFormItem("Reason", reason),
            },
            func(submit bool) {
                if !submit || reason.Text == "" {
                    return
                }
                // TODO: Implement peer reporting
                dialog.ShowInformation("Report Submitted", "The peer has been reported", ui.mainWindow)
            },
            ui.mainWindow,
        )
        reportDialog.Show()
    })

    return container.NewBorder(
        widget.NewCard(
            "Network Status", 
            "", 
            container.NewVBox(
                widget.NewLabel(fmt.Sprintf("Node ID: %s", ui.client.GetNodeID())),
                widget.NewLabel("Connected Peers:"),
            ),
        ),
        container.NewHBox(
            widget.NewButton("Refresh", func() {
                ui.updatePeerList()
            }),
            reportPeer,
        ),
        nil,
        nil,
        container.NewVScroll(ui.peerList),
    )
}

func (ui *FileZapUI) createStorageTab() fyne.CanvasObject {
    // Storage stats
    ui.storageStats = widget.NewLabel("Calculating storage stats...")
    
    // Storage controls
    enableStorage := widget.NewCheck("Enable Storage Node", func(enabled bool) {
        if enabled {
            err := ui.client.EnableStorageNode()
            if err != nil {
                dialog.ShowError(err, ui.mainWindow)
                return
            }
        } else {
            err := ui.client.DisableStorageNode()
            if err != nil {
                dialog.ShowError(err, ui.mainWindow)
                return
            }
        }
        ui.updateStorageStats()
    })

    return container.NewVBox(
        widget.NewCard(
            "Storage Node Status",
            "",
            container.NewVBox(
                enableStorage,
                ui.storageStats,
            ),
        ),
    )
}

func (ui *FileZapUI) createSettingsTab() fyne.CanvasObject {
    storageDir := widget.NewEntry()
    storageDir.SetText(ui.config.StorageDirectory)

    maxStorage := widget.NewEntry()
    maxStorage.SetText(fmt.Sprintf("%d", ui.config.MaxStorageSize/(1024*1024)))

    minSpace := widget.NewEntry()
    minSpace.SetText(fmt.Sprintf("%d", ui.config.MinFreeSpace/(1024*1024)))

    form := &widget.Form{
        Items: []*widget.FormItem{
            {Text: "Storage Directory", Widget: storageDir},
            {Text: "Max Storage (MB)", Widget: maxStorage},
            {Text: "Min Free Space (MB)", Widget: minSpace},
        },
        OnSubmit: func() {
            // Parse and apply settings
            maxSize, err := strconv.ParseInt(maxStorage.Text, 10, 64)
            if err != nil {
                dialog.ShowError(fmt.Errorf("invalid max storage value"), ui.mainWindow)
                return
            }
            minFree, err := strconv.ParseInt(minSpace.Text, 10, 64)
            if err != nil {
                dialog.ShowError(fmt.Errorf("invalid min space value"), ui.mainWindow)
                return
            }

            ui.config.StorageDirectory = storageDir.Text
            ui.config.MaxStorageSize = maxSize * 1024 * 1024  // Convert MB to bytes
            ui.config.MinFreeSpace = minFree * 1024 * 1024    // Convert MB to bytes

            if err := ui.client.UpdateConfig(ui.config); err != nil {
                dialog.ShowError(err, ui.mainWindow)
                return
            }
            dialog.ShowInformation("Settings Saved", "Configuration has been updated", ui.mainWindow)
        },
    }

    return widget.NewCard(
        "Settings",
        "Configure FileZap behavior",
        form,
    )
}

func (ui *FileZapUI) updatePeerList() {
    peers := ui.client.GetPeers()
    ui.peerData = make([]string, len(peers))
    for i, peer := range peers {
        ui.peerData[i] = peer.String()
    }
    ui.peerList.Refresh()
}

func (ui *FileZapUI) updateStorageStats() {
    stats := ui.client.GetStorageStats()
    ui.storageStats.SetText(fmt.Sprintf(
        "Used Space: %d MB / %d MB\n"+
        "Chunks Stored: %d\n"+
        "Storage Requests: %d\n"+
        "Uptime: %.2f%%",
        stats.UsedSpace/(1024*1024),
        stats.MaxSpace/(1024*1024),
        stats.ChunkCount,
        stats.RequestCount,
        stats.Uptime,
    ))
}

func (ui *FileZapUI) Run() {
    ui.mainWindow.Resize(fyne.NewSize(800, 600))
    ui.mainWindow.CenterOnScreen()

    // Start periodic updates
    go ui.periodicUpdates()

    // Cleanup on window close
    ui.mainWindow.SetOnClosed(func() {
        ui.client.Close()
    })

    ui.mainWindow.ShowAndRun()
}

func (ui *FileZapUI) periodicUpdates() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            ui.updatePeerList()
            ui.updateStorageStats()
        case <-ui.client.Context().Done():
            return
        }
    }
}
