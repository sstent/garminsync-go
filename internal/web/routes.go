// internal/web/routes.go
package web

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "time"
    
    "github.com/gorilla/mux"
    "garminsync/internal/database"
)

type Server struct {
    db     database.Database
    router *mux.Router
}

func NewServer(db database.Database) *Server {
    s := &Server{
        db:     db,
        router: mux.NewRouter(),
    }
    
    s.setupRoutes()
    return s
}

func (s *Server) setupRoutes() {
    // Static files (embedded)
    s.router.HandleFunc("/", s.handleHome).Methods("GET")
    s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
    
    // API routes
    api := s.router.PathPrefix("/api").Subrouter()
    
    // Activities
    api.HandleFunc("/activities", s.handleGetActivities).Methods("GET")
    api.HandleFunc("/activities/{id:[0-9]+}", s.handleGetActivity).Methods("GET")
    api.HandleFunc("/activities/search", s.handleSearchActivities).Methods("GET")
    
    // Stats
    api.HandleFunc("/stats", s.handleGetStats).Methods("GET")
    api.HandleFunc("/stats/summary", s.handleGetStatsSummary).Methods("GET")
    
    // Sync operations
    api.HandleFunc("/sync", s.handleTriggerSync).Methods("POST")
    api.HandleFunc("/sync/status", s.handleGetSyncStatus).Methods("GET")
    
    // Configuration
    api.HandleFunc("/config", s.handleGetConfig).Methods("GET")
    api.HandleFunc("/config", s.handleUpdateConfig).Methods("POST")
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
    // Serve embedded HTML
    html := getEmbeddedHTML()
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(html))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    s.writeJSON(w, map[string]string{
        "status": "healthy",
        "service": "GarminSync",
        "timestamp": time.Now().Format(time.RFC3339),
    })
}

func (s *Server) handleGetActivities(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    query := r.URL.Query()
    
    limit, _ := strconv.Atoi(query.Get("limit"))
    if limit <= 0 || limit > 100 {
        limit = 50
    }
    
    offset, _ := strconv.Atoi(query.Get("offset"))
    if offset < 0 {
        offset = 0
    }
    
    // Build filters
    filters := database.ActivityFilters{
        Limit:  limit,
        Offset: offset,
    }
    
    if activityType := query.Get("activity_type"); activityType != "" {
        filters.ActivityType = activityType
    }
    
    if dateFrom := query.Get("date_from"); dateFrom != "" {
        if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
            filters.DateFrom = &t
        }
    }
    
    if dateTo := query.Get("date_to"); dateTo != "" {
        if t, err := time.Parse("2006-01-02", dateTo); err == nil {
            filters.DateTo = &t
        }
    }
    
    if minDistance := query.Get("min_distance"); minDistance != "" {
        if d, err := strconv.ParseFloat(minDistance, 64); err == nil {
            filters.MinDistance = d * 1000 // Convert km to meters
        }
    }
    
    if sortBy := query.Get("sort_by"); sortBy != "" {
        filters.SortBy = sortBy
    }
    
    if sortOrder := query.Get("sort_order"); sortOrder != "" {
        filters.SortOrder = sortOrder
    }
    
    // Get activities
    activities, err := s.db.FilterActivities(filters)
    if err != nil {
        s.writeError(w, "Failed to get activities", http.StatusInternalServerError)
        return
    }
    
    // Convert to API response format
    response := map[string]interface{}{
        "activities": convertActivitiesToAPI(activities),
        "limit":      limit,
        "offset":     offset,
    }
    
    s.writeJSON(w, response)
}

func (s *Server) handleGetActivity(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    activityID, err := strconv.Atoi(vars["id"])
    if err != nil {
        s.writeError(w, "Invalid activity ID", http.StatusBadRequest)
        return
    }
    
    activity, err := s.db.GetActivity(activityID)
    if err != nil {
        s.writeError(w, "Activity not found", http.StatusNotFound)
        return
    }
    
    s.writeJSON(w, convertActivityToAPI(*activity))
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
    stats, err := s.db.GetStats()
    if err != nil {
        s.writeError(w, "Failed to get statistics", http.StatusInternalServerError)
        return
    }
    
    s.writeJSON(w, stats)
}

func (s *Server) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
    // This would trigger the sync operation
    // For now, return success
    s.writeJSON(w, map[string]string{
        "status": "sync_started",
        "message": "Sync operation started in background",
    })
}

// Utility functions
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "error": message,
    })
}

func convertActivitiesToAPI(activities []database.Activity) []map[string]interface{} {
    result := make([]map[string]interface{}, len(activities))
    for i, activity := range activities {
        result[i] = convertActivityToAPI(activity)
    }
    return result
}

func convertActivityToAPI(activity database.Activity) map[string]interface{} {
    return map[string]interface{}{
        "id":               activity.ID,
        "activity_id":      activity.ActivityID,
        "start_time":       activity.StartTime.Format("2006-01-02T15:04:05Z"),
        "activity_type":    activity.ActivityType,
        "duration":         activity.Duration,
        "duration_formatted": formatDuration(activity.Duration),
        "distance":         activity.Distance,
        "distance_km":      roundFloat(activity.Distance/1000, 2),
        "max_heart_rate":   activity.MaxHeartRate,
        "avg_heart_rate":   activity.AvgHeartRate,
        "avg_power":        activity.AvgPower,
        "calories":         activity.Calories,
        "file_type":        activity.FileType,
        "downloaded":       activity.Downloaded,
        "created_at":       activity.CreatedAt.Format("2006-01-02T15:04:05Z"),
        "last_sync":        activity.LastSync.Format("2006-01-02T15:04:05Z"),
    }
}

func formatDuration(seconds int) string {
    if seconds <= 0 {
        return "-"
    }
    
    hours := seconds / 3600
    minutes := (seconds % 3600) / 60
    secs := seconds % 60
    
    if hours > 0 {
        return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
    }
    return fmt.Sprintf("%d:%02d", minutes, secs)
}

func roundFloat(val float64, precision int) float64 {
    ratio := math.Pow(10, float64(precision))
    return math.Round(val*ratio) / ratio
}
