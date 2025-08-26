package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"strings"

	"github.com/sstent/garminsync-go/internal/database"
	"github.com/sstent/garminsync-go/internal/garmin"
	"github.com/sstent/garminsync-go/internal/parser"
)

type SyncService struct {
	garminClient *garmin.Client
	db           *database.SQLiteDB
	dataDir      string
}

func NewSyncService(garminClient *garmin.Client, db *database.SQLiteDB, dataDir string) *SyncService {
	return &SyncService{
		garminClient: garminClient,
		db:           db,
		dataDir:      dataDir,
	}
}

func (s *SyncService) testAPIConnectivity() error {
    // Try a simple API call to check connectivity
    _, err := s.garminClient.GetActivities(0, 1)
    if err != nil {
        // Analyze error for troubleshooting hints
        if strings.Contains(err.Error(), "connection refused") {
            return fmt.Errorf("API connection failed: service might not be running. Verify garmin-api container is up. Original error: %w", err)
        } else if strings.Contains(err.Error(), "timeout") {
            return fmt.Errorf("API connection timeout: service might be slow to start. Original error: %w", err)
        } else if strings.Contains(err.Error(), "status 5") {
            return fmt.Errorf("API server error: check garmin-api logs. Original error: %w", err)
        }
        return fmt.Errorf("API connectivity test failed: %w", err)
    }
    return nil
}

func (s *SyncService) FullSync(ctx context.Context) error {
    fmt.Println("=== Starting full sync ===")
    defer fmt.Println("=== Sync completed ===")
    
    // Check API connectivity before proceeding
    if err := s.testAPIConnectivity(); err != nil {
        return fmt.Errorf("API connectivity test failed: %w", err)
    }
    fmt.Println("✅ API connectivity verified")

	// Check credentials first
	email := os.Getenv("GARMIN_EMAIL")
	password := os.Getenv("GARMIN_PASSWORD")
	
	if email == "" || password == "" {
        errorMsg := fmt.Sprintf("Missing credentials - GARMIN_EMAIL: '%s', GARMIN_PASSWORD: %s", 
            email, 
            map[bool]string{true: "SET", false: "EMPTY"}[password != ""])
        errorMsg += "\nTroubleshooting:"
        errorMsg += "\n1. Ensure the .env file exists with GARMIN_EMAIL and GARMIN_PASSWORD"
        errorMsg += "\n2. Verify docker-compose.yml mounts the .env file"
        errorMsg += "\n3. Check container env vars: docker-compose exec garminsync env | grep GARMIN"
        return fmt.Errorf(errorMsg)
	}
	
	fmt.Printf("Using credentials - Email: %s, Password: %s\n", email, 
		map[bool]string{true: "***SET***", false: "EMPTY"}[password != ""])

	// 1. Fetch activities from Garmin
	fmt.Println("Fetching activities from Garmin Connect...")
	activities, err := s.garminClient.GetActivities(0, 10) // Start with just 10 for testing
	if err != nil {
		return fmt.Errorf("failed to get activities: %w", err)
	}
	
	fmt.Printf("✅ Found %d activities from Garmin\n", len(activities))
	
	if len(activities) == 0 {
		fmt.Println("⚠️ No activities returned - this might be expected if:")
		fmt.Println("   - Your Garmin account has no activities")
		fmt.Println("   - The API response format changed")
		fmt.Println("   - Authentication succeeded but data access failed")
		return nil
	}

	// 2. Process each activity
	for i, activity := range activities {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Printf("[%d/%d] Processing activity %d (%s)...\n", 
				i+1, len(activities), activity.ActivityID, activity.ActivityName)
			if err := s.syncActivity(&activity); err != nil {
				fmt.Printf("❌ Error syncing activity %d: %v\n", activity.ActivityID, err)
			} else {
				fmt.Printf("✅ Successfully synced activity %d\n", activity.ActivityID)
			}
		}
	}

	return nil
}

func (s *SyncService) syncActivity(activity *garmin.GarminActivity) error {
	// Skip if already downloaded
	if exists, _ := s.db.ActivityExists(activity.ActivityID); exists {
		return nil
	}

	// Download the activity file (FIT format)
	fileData, err := s.garminClient.DownloadActivity(activity.ActivityID, "fit")
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Save file
	filename := filepath.Join(s.dataDir, "activities", fmt.Sprintf("%d.fit", activity.ActivityID))
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("directory creation failed: %w", err)
	}
	if err := os.WriteFile(filename, fileData, 0644); err != nil {
		return fmt.Errorf("file write failed: %w", err)
	}

	// Parse the file
	fileParser := parser.NewParser()
	metrics, err := fileParser.ParseData(fileData)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	// Parse start time
	startTime, err := time.Parse("2006-01-02 15:04:05", activity.StartTimeLocal)
	if err != nil {
		startTime = time.Now()
	}

	// Save to database
	if err := s.db.CreateActivity(&database.Activity{
		ActivityID:    activity.ActivityID,
		StartTime:     startTime,
		ActivityType:  getActivityType(activity),
		Distance:      metrics.Distance,
		Duration:      int(metrics.Duration.Seconds()),
		MaxHeartRate:  metrics.MaxHeartRate,
		AvgHeartRate:  metrics.AvgHeartRate,
		AvgPower:      float64(metrics.AvgPower),
		Calories:      metrics.Calories,
		Filename:      filename,
		FileType:      "fit",
		Downloaded:    true,
		ElevationGain: metrics.ElevationGain,
		Steps:         metrics.Steps,
	}); err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	fmt.Printf("Synced activity %d\n", activity.ActivityID)
	return nil
}

// Add missing Sync method
func (s *SyncService) Sync(ctx context.Context) error {
    return s.FullSync(ctx)
}

func getActivityType(activity *garmin.GarminActivity) string {
	if activityType, ok := activity.ActivityType["typeKey"]; ok {
		return activityType.(string)
	}
	return "unknown"
}
