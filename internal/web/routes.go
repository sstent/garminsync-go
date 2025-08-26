package web

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sstent/garminsync-go/internal/database"
	"github.com/sstent/garminsync-go/internal/garmin"
	"github.com/sstent/garminsync-go/internal/sync"
)

type WebHandler struct {
	db       *database.SQLiteDB
	syncer   *sync.SyncService
	garmin   *garmin.Client
}

func NewWebHandler(db *database.SQLiteDB, syncer *sync.SyncService, garmin *garmin.Client) *WebHandler {
	return &WebHandler{
		db:       db,
		syncer:   syncer,
		garmin:   garmin,
	}
}

func (h *WebHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/stats", h.GetStats)
	router.GET("/activities", h.ActivityList)
	router.GET("/activities/:id", h.ActivityDetail)
	router.POST("/sync", h.Sync)
}

func (h *WebHandler) GetStats(c *gin.Context) {
	stats, err := h.db.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *WebHandler) ActivityList(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	
	if limit <= 0 {
		limit = 50
	}
	
	activities, err := h.db.GetActivities(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activities"})
		return
	}
	
	c.JSON(http.StatusOK, activities)
}

func (h *WebHandler) ActivityDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}
	
	activity, err := h.db.GetActivity(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}
	
	c.JSON(http.StatusOK, activity)
}

func (h *WebHandler) Sync(c *gin.Context) {
	go func() {
		err := h.syncer.Sync(context.Background())
		if err != nil {
			log.Printf("Sync error: %v", err)
		}
	}()
	
	c.JSON(http.StatusOK, gin.H{"status": "sync_started", "message": "Sync started in background"})
}
