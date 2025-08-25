// main.go - Entry point and dependency injection
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/yourusername/garminsync/internal/database"
	"github.com/yourusername/garminsync/internal/garmin"
	"github.com/yourusername/garminsync/internal/sync"
	"github.com/yourusername/garminsync/internal/web"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
)

type App struct {
	db         *database.SQLiteDB
	cron       *cron.Cron
	server     *http.Server
	garmin     *garmin.Client
	shutdown   chan os.Signal
	syncService *sync.SyncService
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

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
	dbConn, err := initDatabase()
	if err != nil {
		return err
	}
	app.db = database.NewSQLiteDBFromDB(dbConn)

	// Initialize Garmin client
	app.garmin = garmin.NewClient()

	// Initialize sync service
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	app.syncService = sync.NewSyncService(app.garmin, database.NewSQLiteDBFromDB(app.db), dataDir)

	// Setup cron scheduler
	app.cron = cron.New()

	// Setup HTTP server
	webHandler := web.NewWebHandler(app.db)
	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "./internal/web/templates"
	}
	if err := webHandler.LoadTemplates(templateDir); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	app.server = &http.Server{
		Addr:    ":8888",
		Handler: app.setupRoutes(webHandler),
	}

	return nil
}

func (app *App) start() {
	// Start cron scheduler
	app.cron.AddFunc("@hourly", func() {
		log.Println("Starting scheduled sync...")
		if err := app.syncService.Sync(context.Background()); err != nil {
			log.Printf("Sync failed: %v", err)
		}
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

// Database initialization
func initDatabase() (*sql.DB, error) {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		// Fallback to DATA_DIR/garmin.db if DB_PATH not set
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		
		// Create data directory if it doesn't exist
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %v", err)
		}
		
		dbPath = filepath.Join(dataDir, "garmin.db")
	}
	
	// Initialize SQLite database
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	
	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %v", err)
	}
	
	// Create tables if they don't exist
	sqliteDB := &database.SQLiteDB{db: db}
	if err := sqliteDB.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return db, nil
}

// Application routes
func (app *App) setupRoutes(webHandler *web.WebHandler) *http.ServeMux {
	mux := http.NewServeMux()
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Web UI routes
	mux.HandleFunc("/", webHandler.Index)
	mux.HandleFunc("/activities", webHandler.ActivityList)
	mux.HandleFunc("/activity", webHandler.ActivityDetail)
	
	return mux
}
