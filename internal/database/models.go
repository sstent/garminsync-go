// internal/database/models.go
package database

import (
    "database/sql"
    "time"
)

type Activity struct {
    ID            int       `json:"id"`
    ActivityID    int       `json:"activity_id"`
    StartTime     time.Time `json:"start_time"`
    ActivityType  string    `json:"activity_type"`
    Duration      int       `json:"duration"`          // seconds
    Distance      float64   `json:"distance"`          // meters
    MaxHeartRate  int       `json:"max_heart_rate"`
    AvgHeartRate  int       `json:"avg_heart_rate"`
    AvgPower      float64   `json:"avg_power"`
    Calories      int       `json:"calories"`
    Filename      string    `json:"filename"`
    FileType      string    `json:"file_type"`
    FileSize      int64     `json:"file_size"`
    Downloaded    bool      `json:"downloaded"`
    CreatedAt     time.Time `json:"created_at"`
    LastSync      time.Time `json:"last_sync"`
}

type Stats struct {
    Total      int `json:"total"`
    Downloaded int `json:"downloaded"`
    Missing    int `json:"missing"`
}

type DaemonConfig struct {
    ID           int    `json:"id"`
    Enabled      bool   `json:"enabled"`
    ScheduleCron string `json:"schedule_cron"`
    LastRun      string `json:"last_run"`
    Status       string `json:"status"`
}

// Database interface
type Database interface {
    // Activities
    GetActivities(limit, offset int) ([]Activity, error)
    GetActivity(activityID int) (*Activity, error)
    CreateActivity(activity *Activity) error
    UpdateActivity(activity *Activity) error
    DeleteActivity(activityID int) error
    
    // Stats
    GetStats() (*Stats, error)
    
    // Search and filter
    FilterActivities(filters ActivityFilters) ([]Activity, error)
    
    // Close connection
    Close() error
}

type ActivityFilters struct {
    ActivityType string
    DateFrom     *time.Time
    DateTo       *time.Time
    MinDistance  float64
    MaxDistance  float64
    MinDuration  int
    MaxDuration  int
    Downloaded   *bool
    Limit        int
    Offset       int
    SortBy       string
    SortOrder    string
}
