package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

func (s *SyncService) FullSync(ctx context.Context) error {
	fmt.Println("Starting full sync...")
	defer fmt.Println("Sync completed")

	// 1. Fetch activities from Garmin
	activities, err := s.garminClient.GetActivities(0, 100)
	if err != nil {
		return fmt.Errorf("failed to get activities: %w", err)
	}
	fmt.Printf("Found %d activities\n", len(activities))

	// 2. Process each activity
	for _, activity := range activities {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Printf("Processing activity %d...\n", activity.ActivityID)
			if err := s.syncActivity(&activity); err != nil {
				fmt.Printf("Error syncing activity: %v\n", err)
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
