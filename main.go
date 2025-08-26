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

	"github.com/sstent/garminsync-go/internal/database"
	"github.com/sstent/garminsync-go/internal/garmin"
	"github.com/sstent/garminsync-go/internal/sync"
	"github.com/sstent/garminsync-go/internal/web"

	_ "github.com/mattn/go-sqlite3"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type App struct {
	db         *database.SQLiteDB
	cron       *cron.Cron
	server     *http.Server
	garmin     *garmin.Client
	shutdown   chan os.Signal
	syncService *sync.SyncService  // This should now work
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
	app.db, err = initDatabase()
	if err != nil {
		return err
	}

	// Initialize Garmin client
	app.garmin = garmin.NewClient()

	// Initialize sync service
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	app.syncService = sync.NewSyncService(app.garmin, app.db, dataDir)

	// Setup cron scheduler
	app.cron = cron.New()

	// Setup HTTP server
	webHandler := web.NewWebHandler(app.db, app.syncService, app.garmin)
	// We've removed template loading since we're using static frontend
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
func initDatabase() (*database.SQLiteDB, error) {
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
	sqliteDB := database.NewSQLiteDBFromDB(db)
	if err := sqliteDB.CreateTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return sqliteDB, nil
}

func (app *App) setupRoutes(webHandler *web.WebHandler) http.Handler {
	router := gin.Default()
	
	// Add middleware
	router.Use(gin.Logger())   // Log all requests
	router.Use(gin.Recovery()) // Recover from any panics

	// Enable CORS for development
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	
	// Serve static files
	router.Static("/static", "./web/static")
	router.LoadHTMLFiles("web/index.html")
	
	// API routes
	api := router.Group("/api")
	webHandler.RegisterRoutes(api)
	
	// Serve main page
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	
	return router
}
