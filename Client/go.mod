module github.com/VetheonGames/FileZap/Client

go 1.21

require (
    fyne.io/fyne/v2 v2.6.1
    github.com/VetheonGames/FileZap/Divider v0.0.0-00010101000000-000000000000
    github.com/VetheonGames/FileZap/NetworkCore v0.0.0-00010101000000-000000000000
)

replace (
    github.com/VetheonGames/FileZap/Divider => ./../Divider
    github.com/VetheonGames/FileZap/NetworkCore => "./../Network Core"
)
