package ui

import (
    "context"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/storage"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
    "github.com/VetheonGames/FileZap/Client/pkg/operations"
    "github.com/VetheonGames/FileZap/Client/pkg/server"
    "path/filepath"
)

type FileZapUI struct {
    app        fyne.App
    mainWindow fyne.Window
    server     *server.IntegratedServer
    fileOps    *operations.FileOperations
    dataDir    string
}

func NewFileZapUI() *FileZapUI {
    ui := &FileZapUI{
        app:     app.New(),
        dataDir: filepath.Join(".", "data"),
    }
    
    // Initialize integrated server
    srv, err := server.NewIntegratedServer(context.Background(), ui.dataDir)
    if err != nil {
        panic(err)
    }
    ui.server = srv
    
    ui.fileOps = operations.NewFileOperations(ui.server)
    ui.mainWindow = ui.app.NewWindow("FileZap")
    ui.setupUI()
    
    // Start server
    if err := ui.server.Start(); err != nil {
        dialog.ShowError(err, ui.mainWindow)
    }
    
    return ui
}

func (ui *FileZapUI) setupUI() {
    // Create tabs for different operations
    tabs := container.NewAppTabs(
        container.NewTabItem("Split File", ui.createSplitTab()),
        container.NewTabItem("Join File", ui.createJoinTab()),
        container.NewTabItem("Network", ui.createNetworkTab()),
    )
    tabs.SetTabLocation(container.TabLocationTop)

    ui.mainWindow.SetContent(tabs)
}

func (ui *FileZapUI) createSplitTab() fyne.CanvasObject {
    inputPath := widget.NewEntry()
    inputPath.SetPlaceHolder("Input File Path")

    outputPath := widget.NewEntry()
    outputPath.SetPlaceHolder("Output Directory")

    chunkSize := widget.NewEntry()
    chunkSize.SetText("1048576") // Default 1MB chunks

    inputSelect := widget.NewButton("Select Input", func() {
        dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
            if err != nil || file == nil {
                return
            }
            inputPath.SetText(file.URI().Path())
        }, ui.mainWindow)
        dialog.Show()
    })

    outputSelect := widget.NewButton("Select Output", func() {
        dialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
            if err != nil || uri == nil {
                return
            }
            outputPath.SetText(uri.Path())
        }, ui.mainWindow)
        dialog.Show()
    })

    splitButton := widget.NewButton("Split File", func() {
        if err := ui.fileOps.SplitFile(inputPath.Text, outputPath.Text, chunkSize.Text); err != nil {
            dialog.ShowError(err, ui.mainWindow)
        }
    })
    splitButton.Icon = theme.ContentCutIcon()

    return container.NewVBox(
        widget.NewLabel("Split File into Chunks"),
        container.NewBorder(nil, nil, nil, inputSelect, inputPath),
        container.NewBorder(nil, nil, nil, outputSelect, outputPath),
        widget.NewLabel("Chunk Size (bytes):"),
        chunkSize,
        splitButton,
    )
}

func (ui *FileZapUI) createJoinTab() fyne.CanvasObject {
    zapPath := widget.NewEntry()
    zapPath.SetPlaceHolder("ZAP File Path")

    outputPath := widget.NewEntry()
    outputPath.SetPlaceHolder("Output Directory")

    zapSelect := widget.NewButton("Select ZAP", func() {
        dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
            if err != nil || file == nil {
                return
            }
            zapPath.SetText(file.URI().Path())
        }, ui.mainWindow)
        dialog.SetFilter(storage.NewExtensionFileFilter([]string{".zap"}))
        dialog.Show()
    })

    outputSelect := widget.NewButton("Select Output", func() {
        dialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
            if err != nil || uri == nil {
                return
            }
            outputPath.SetText(uri.Path())
        }, ui.mainWindow)
        dialog.Show()
    })

    joinButton := widget.NewButton("Join File", func() {
        if err := ui.fileOps.JoinFile(zapPath.Text, outputPath.Text); err != nil {
            dialog.ShowError(err, ui.mainWindow)
        }
    })
    joinButton.Icon = theme.ContentPasteIcon()

    return container.NewVBox(
        widget.NewLabel("Join File from Chunks"),
        container.NewBorder(nil, nil, nil, zapSelect, zapPath),
        container.NewBorder(nil, nil, nil, outputSelect, outputPath),
        joinButton,
    )
}

func (ui *FileZapUI) createNetworkTab() fyne.CanvasObject {
    nodeID := widget.NewLabel("Node ID: " + ui.server.GetNodeID())
    
    peerList := widget.NewTextGrid()
    updatePeers := func() {
        peers := ui.server.GetPeers()
        text := "Connected Peers:\n"
        for _, peer := range peers {
            text += "- " + peer + "\n"
        }
        peerList.SetText(text)
    }
    updatePeers() // Initial update
    
    refreshBtn := widget.NewButton("Refresh Peers", updatePeers)
    
    return container.NewVBox(
        nodeID,
        widget.NewLabel("Network Status"),
        peerList,
        refreshBtn,
    )
}

func (ui *FileZapUI) Run() {
    ui.mainWindow.Resize(fyne.NewSize(800, 600))
    ui.mainWindow.CenterOnScreen()
    
    // Cleanup on window close
    ui.mainWindow.SetOnClosed(func() {
        if err := ui.server.Stop(); err != nil {
            dialog.ShowError(err, ui.mainWindow)
        }
    })
    
    ui.mainWindow.ShowAndRun()
}
