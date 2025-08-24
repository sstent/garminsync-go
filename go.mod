// go.mod - Keep dependencies minimal
module garminsync

go 1.21

require (
    github.com/mattn/go-sqlite3 v1.14.17
    github.com/robfig/cron/v3 v3.0.1
    github.com/gorilla/mux v1.8.0       // For HTTP routing
    golang.org/x/net v0.12.0            // For HTTP client
)
