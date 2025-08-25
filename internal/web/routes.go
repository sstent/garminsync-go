package web

import (
	"context"
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
	templates map[string]interface{} // Placeholder for template handling
}

func NewWebHandler(db *database.SQLiteDB, syncer *sync.SyncService, garmin *garmin.Client) *WebHandler {
	return &WebHandler{
		db:       db,
		syncer:   syncer,
		garmin:   garmin,
		templates: make(map[string]interface{}),
	}
}

func (h *WebHandler) LoadTemplates(templateDir string) error {
	// For now, just return nil - templates will be handled later
	return nil
}

func (h *WebHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/", h.Index)
	router.GET("/activities", h.ActivityList)
	router.GET("/activities/:id", h.ActivityDetail)
	router.POST("/sync", h.Sync)
}

func (h *WebHandler) Index(c *gin.Context) {
	stats, err := h.db.GetStats()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	
	// Placeholder for template rendering
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
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	
	c.JSON(http.StatusOK, activities)
}

func (h *WebHandler) ActivityDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	
	activity, err := h.db.GetActivity(id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	
	c.JSON(http.StatusOK, activity)
}

func (h *WebHandler) Sync(c *gin.Context) {
	err := h.syncer.Sync(context.Background())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	
	c.Status(http.StatusOK)
}
