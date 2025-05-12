module github.com/VetheonGames/FileZap/Validator-Server

go 1.21

require (
    github.com/gorilla/mux v1.8.1
    github.com/VetheonGames/FileZap/NetworkCore v0.0.0
)

replace github.com/VetheonGames/FileZap/NetworkCore => "../Network Core"
