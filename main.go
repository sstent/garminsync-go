// main.go - Entry point and dependency injection
package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    _ "github.com/mattn/go-sqlite3"
    "github.com/robfig/cron/v3"
)

type App struct {
    db       *sql.DB
    cron     *cron.Cron
    server   *http.Server
    garmin   *GarminClient
    shutdown chan os.Signal
}

func main() {
    app := &App{
        shutdown: make(chan os.Signal, 1),
    }
    
    // Initialize components
    if err := app.init(); err != nil {
        log.Fatal("Failed to initialize app:", err)
    }
    
    // Start services
    app.start()
    
    // Wait for shutdown signal
    signal.Notify(app.shutdown, os.Interrupt, syscall.SIGTERM)
    <-app.shutdown
    
    // Graceful shutdown
    app.stop()
}

func (app *App) init() error {
    var err error
    
    // Initialize database
    app.db, err = initDatabase()
    if err != nil {
        return err
    }
    
    // Initialize Garmin client
    app.garmin = NewGarminClient()
    
    // Setup cron scheduler
    app.cron = cron.New()
    
    // Setup HTTP server
    app.server = &http.Server{
        Addr:    ":8888",
        Handler: app.setupRoutes(),
    }
    
    return nil
}

func (app *App) start() {
    // Start cron scheduler
    app.cron.AddFunc("@hourly", func() {
        log.Println("Starting scheduled sync...")
        app.syncActivities()
    })
    app.cron.Start()
    
    // Start web server
    go func() {
        log.Println("Server starting on http://localhost:8888")
        if err := app.server.ListenAndServe(); err != http.ErrServerClosed {
            log.Printf("Server error: %v", err)
        }
    }()
}

func (app *App) stop() {
    log.Println("Shutting down...")
    
    // Stop cron
    app.cron.Stop()
    
    // Stop web server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := app.server.Shutdown(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }
    
    // Close database
    if app.db != nil {
        app.db.Close()
    }
    
    log.Println("Shutdown complete")
}

// main.go - Database initialization
func initDatabase() (*sql.DB, error) {
    // Get data directory from environment or use default
    dataDir := os.Getenv("DATA_DIR")
    if dataDir == "" {
        dataDir = "./data"
    }
    
    // Create data directory if it doesn't exist
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create data directory: %v", err)
    }
    
    dbPath := filepath.Join(dataDir, "garmin.db")
    
    // Initialize SQLite database
    db, err := database.NewSQLiteDB(dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize database: %v", err)
    }
    
    return db.db, nil // Return the underlying *sql.DB
}
