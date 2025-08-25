// internal/database/sqlite.go
package database

import (
    "database/sql"
    "fmt"
    "strings"
    "time"
)

type SQLiteDB struct {
    db *sql.DB
}

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
    db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
    if err != nil {
        return nil, err
    }
    
    sqlite := &SQLiteDB{db: db}
    
    // Create tables
    if err := sqlite.createTables(); err != nil {
        return nil, err
    }
    
    return sqlite, nil
}

func (s *SQLiteDB) createTables() error {
    schema := `
	CREATE TABLE IF NOT EXISTS activities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		activity_id INTEGER UNIQUE NOT NULL,
		start_time DATETIME NOT NULL,
		activity_type TEXT,
		duration INTEGER,
		distance REAL,
		max_heart_rate INTEGER,
		avg_heart_rate INTEGER,
		avg_power REAL,
		calories INTEGER,
		steps INTEGER,
		elevation_gain REAL,
		start_latitude REAL,
		start_longitude REAL,
		filename TEXT UNIQUE,
		file_type TEXT,
		file_size INTEGER,
		downloaded BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_sync DATETIME DEFAULT CURRENT_TIMESTAMP
	);
    
    CREATE INDEX IF NOT EXISTS idx_activities_activity_id ON activities(activity_id);
    CREATE INDEX IF NOT EXISTS idx_activities_start_time ON activities(start_time);
    CREATE INDEX IF NOT EXISTS idx_activities_activity_type ON activities(activity_type);
    CREATE INDEX IF NOT EXISTS idx_activities_downloaded ON activities(downloaded);
    
    CREATE TABLE IF NOT EXISTS daemon_config (
        id INTEGER PRIMARY KEY DEFAULT 1,
        enabled BOOLEAN DEFAULT TRUE,
        schedule_cron TEXT DEFAULT '0 * * * *',
        last_run TEXT,
        status TEXT DEFAULT 'stopped',
        CONSTRAINT single_config CHECK (id = 1)
    );
    
    INSERT OR IGNORE INTO daemon_config (id) VALUES (1);
    `
    
    _, err := s.db.Exec(schema)
    return err
}

func (s *SQLiteDB) GetActivities(limit, offset int) ([]Activity, error) {
    query := `
    SELECT id, activity_id, start_time, activity_type, duration, distance, 
           max_heart_rate, avg_heart_rate, avg_power, calories, steps, 
           elevation_gain, start_latitude, start_longitude,
           filename, file_type, file_size, downloaded, created_at, last_sync
    FROM activities 
    ORDER BY start_time DESC 
    LIMIT ? OFFSET ?`
    
    rows, err := s.db.Query(query, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var activities []Activity
    for rows.Next() {
        var a Activity
        var startTime, createdAt, lastSync string
        
        err := rows.Scan(
            &a.ID, &a.ActivityID, &startTime, &a.ActivityType,
            &a.Duration, &a.Distance, &a.MaxHeartRate, &a.AvgHeartRate,
            &a.AvgPower, &a.Calories, &a.Steps, &a.ElevationGain,
            &a.StartLatitude, &a.StartLongitude,
            &a.Filename, &a.FileType, &a.FileSize, &a.Downloaded,
            &createdAt, &lastSync,
        )
        if err != nil {
            return nil, err
        }
        
        // Parse time strings
        if a.StartTime, err = time.Parse("2006-01-02 15:04:05", startTime); err != nil {
            return nil, err
        }
        if a.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt); err != nil {
            return nil, err
        }
        if a.LastSync, err = time.Parse("2006-01-02 15:04:05", lastSync); err != nil {
            return nil, err
        }
        
        activities = append(activities, a)
    }
    
    return activities, nil
}

func (s *SQLiteDB) ActivityExists(activityID int) (bool, error) {
	query := `SELECT COUNT(*) FROM activities WHERE activity_id = ?`
	var count int
	err := s.db.QueryRow(query, activityID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteDB) GetActivity(activityID int) (*Activity, error) {
    query := `
    SELECT id, activity_id, start_time, activity_type, duration, distance, 
           max_heart_rate, avg_heart_rate, avg_power, calories, steps, 
           elevation_gain, start_latitude, start_longitude,
           filename, file_type, file_size, downloaded, created_at, last_sync
    FROM activities 
    WHERE activity_id = ?`
    
    row := s.db.QueryRow(query, activityID)
    
    var a Activity
    var startTime, createdAt, lastSync string
    
    err := row.Scan(
        &a.ID, &a.ActivityID, &startTime, &a.ActivityType,
        &a.Duration, &a.Distance, &a.MaxHeartRate, &a.AvgHeartRate,
        &a.AvgPower, &a.Calories, &a.Steps, &a.ElevationGain,
        &a.StartLatitude, &a.StartLongitude,
        &a.Filename, &a.FileType, &a.FileSize, &a.Downloaded,
        &createdAt, &lastSync,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("activity not found")
        }
        return nil, err
    }
    
    // Parse time strings
    if a.StartTime, err = time.Parse("2006-01-02 15:04:05", startTime); err != nil {
        return nil, err
    }
    if a.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt); err != nil {
        return nil, err
    }
    if a.LastSync, err = time.Parse("2006-01-02 15:04:05", lastSync); err != nil {
        return nil, err
    }
    
    return &a, nil
}

func (s *SQLiteDB) CreateActivity(activity *Activity) error {
	query := `
	INSERT INTO activities (
		activity_id, start_time, activity_type, duration, distance,
		max_heart_rate, avg_heart_rate, avg_power, calories,
		steps, elevation_gain, start_latitude, start_longitude,
		filename, file_type, file_size, downloaded
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    
    _, err := s.db.Exec(query,
	activity.ActivityID, activity.StartTime.Format("2006-01-02 15:04:05"),
	activity.ActivityType, activity.Duration, activity.Distance,
	activity.MaxHeartRate, activity.AvgHeartRate, activity.AvgPower,
	activity.Calories, activity.Steps, activity.ElevationGain,
	activity.StartLatitude, activity.StartLongitude,
	activity.Filename, activity.FileType,
	activity.FileSize, activity.Downloaded,
    )
    
    return err
}

func (s *SQLiteDB) UpdateActivity(activity *Activity) error {
	query := `
	UPDATE activities SET 
		activity_type = ?, duration = ?, distance = ?,
		max_heart_rate = ?, avg_heart_rate = ?, avg_power = ?,
		calories = ?, steps = ?, elevation_gain = ?,
		start_latitude = ?, start_longitude = ?,
		filename = ?, file_type = ?, file_size = ?,
		downloaded = ?, last_sync = CURRENT_TIMESTAMP
	WHERE activity_id = ?`
    
    _, err := s.db.Exec(query,
		activity.ActivityType, activity.Duration, activity.Distance,
		activity.MaxHeartRate, activity.AvgHeartRate, activity.AvgPower,
		activity.Calories, activity.Steps, activity.ElevationGain,
		activity.StartLatitude, activity.StartLongitude,
		activity.Filename, activity.FileType,
		activity.FileSize, activity.Downloaded, activity.ActivityID,
    )
    
    return err
}

func (s *SQLiteDB) GetStats() (*Stats, error) {
    stats := &Stats{}
    
    // Get total count
    err := s.db.QueryRow("SELECT COUNT(*) FROM activities").Scan(&stats.Total)
    if err != nil {
        return nil, err
    }
    
    // Get downloaded count
    err = s.db.QueryRow("SELECT COUNT(*) FROM activities WHERE downloaded = TRUE").Scan(&stats.Downloaded)
    if err != nil {
        return nil, err
    }
    
    stats.Missing = stats.Total - stats.Downloaded
    
    return stats, nil
}

func (s *SQLiteDB) FilterActivities(filters ActivityFilters) ([]Activity, error) {
    query := `
	SELECT id, activity_id, start_time, activity_type, duration, distance, 
		   max_heart_rate, avg_heart_rate, avg_power, calories, steps, 
		   elevation_gain, start_latitude, start_longitude,
		   filename, file_type, file_size, downloaded, created_at, last_sync
    FROM activities WHERE 1=1`
    
    var args []interface{}
    var conditions []string
    
    // Build WHERE conditions
    if filters.ActivityType != "" {
        conditions = append(conditions, "activity_type = ?")
        args = append(args, filters.ActivityType)
    }
    
    if filters.DateFrom != nil {
        conditions = append(conditions, "start_time >= ?")
        args = append(args, filters.DateFrom.Format("2006-01-02 15:04:05"))
    }
    
    if filters.DateTo != nil {
        conditions = append(conditions, "start_time <= ?")
        args = append(args, filters.DateTo.Format("2006-01-02 15:04:05"))
    }
    
    if filters.MinDistance > 0 {
        conditions = append(conditions, "distance >= ?")
        args = append(args, filters.MinDistance)
    }
    
    if filters.MaxDistance > 0 {
        conditions = append(conditions, "distance <= ?")
        args = append(args, filters.MaxDistance)
    }
    
    if filters.Downloaded != nil {
        conditions = append(conditions, "downloaded = ?")
        args = append(args, *filters.Downloaded)
    }
    
    // Add conditions to query
    if len(conditions) > 0 {
        query += " AND " + strings.Join(conditions, " AND ")
    }
    
    // Add sorting
    orderBy := "start_time"
    if filters.SortBy != "" {
        orderBy = filters.SortBy
    }
    
    order := "DESC"
    if filters.SortOrder == "asc" {
        order = "ASC"
    }
    
    query += fmt.Sprintf(" ORDER BY %s %s", orderBy, order)
    
    // Add pagination
    if filters.Limit > 0 {
        query += " LIMIT ?"
        args = append(args, filters.Limit)
        
        if filters.Offset > 0 {
            query += " OFFSET ?"
            args = append(args, filters.Offset)
        }
    }
    
    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var activities []Activity
    for rows.Next() {
        var a Activity
        var startTime, createdAt, lastSync string
        
        err := rows.Scan(
		&a.ID, &a.ActivityID, &startTime, &a.ActivityType,
		&a.Duration, &a.Distance, &a.MaxHeartRate, &a.AvgHeartRate,
		&a.AvgPower, &a.Calories, &a.Steps, &a.ElevationGain,
		&a.StartLatitude, &a.StartLongitude,
		&a.Filename, &a.FileType, &a.FileSize, &a.Downloaded,
		&createdAt, &lastSync,
        )
        if err != nil {
            return nil, err
        }
        
        // Parse times
        a.StartTime, _ = time.Parse("2006-01-02 15:04:05", startTime)
        a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
        a.LastSync, _ = time.Parse("2006-01-02 15:04:05", lastSync)
        
        activities = append(activities, a)
    }
    
    return activities, nil
}

func (s *SQLiteDB) Close() error {
    return s.db.Close()
}

// NewSQLiteDBFromDB wraps an existing sql.DB connection
func NewSQLiteDBFromDB(db *sql.DB) *SQLiteDB {
	return &SQLiteDB{db: db}
}
