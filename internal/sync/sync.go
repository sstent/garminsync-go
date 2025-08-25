package sync

import (
	"context"
	"fmt"
	"io"
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

func (s *SyncService) Sync(ctx context.Context) error {
	startTime := time.Now()
	fmt.Printf("Starting sync at %s\n", startTime.Format(time.RFC3339))
	defer func() {
		fmt.Printf("Sync completed in %s\n", time.Since(startTime))
	}()

	// 1. Fetch latest activities from Garmin
	activities, err := s.garminClient.GetActivities(0, 100)
	if err != nil {
		return fmt.Errorf("failed to get activities: %w", err)
	}
	fmt.Printf("Found %d activities on Garmin\n", len(activities))

	// 2. Sync each activity
	for i, activity := range activities {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Printf("[%d/%d] Processing activity %d...\n", i+1, len(activities), activity.ActivityID)
			if err := s.syncActivity(&activity); err != nil {
				fmt.Printf("Error syncing activity %d: %v\n", activity.ActivityID, err)
				// Continue with next activity on error
			}
		}
	}

	return nil
}

func (s *SyncService) syncActivity(activity *garmin.GarminActivity) error {
	// Check if activity exists in database
	dbActivity, err := s.db.GetActivity(activity.ActivityID)
	if err == nil {
		// Activity exists - check if already downloaded
		if dbActivity.Downloaded {
			fmt.Printf("Activity %d already downloaded\n", activity.ActivityID)
			return nil
		}
	} else {
		// Activity not in database - create new record
		dbActivity = &database.Activity{
			ActivityID: activity.ActivityID,
			StartTime:  parseTime(activity.StartTimeLocal),
		}
		
		// Add basic info if available
		if activityType, ok := activity.ActivityType["typeKey"]; ok {
			dbActivity.ActivityType = activityType.(string)
		}
		dbActivity.Duration = int(activity.Duration)
		dbActivity.Distance = activity.Distance
		
		if err := s.db.CreateActivity(dbActivity); err != nil {
			return fmt.Errorf("failed to create activity: %w", err)
		}
	}

	// Download the activity file (FIT format)
	fileData, err := s.garminClient.DownloadActivity(activity.ActivityID, "fit")
	if err != nil {
		return fmt.Errorf("failed to download activity: %w", err)
	}

	// Determine filename
	filename := filepath.Join(
		s.dataDir,
		"activities",
		fmt.Sprintf("%d_%s.fit", activity.ActivityID, activity.StartTimeLocal[:10]),
	)

	// Create directories if needed
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save file
	if err := os.WriteFile(filename, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Parse the file to extract additional metrics
	metrics, err := parser.NewFITParser().ParseData(fileData)
	if err != nil {
		return fmt.Errorf("failed to parse activity file: %w", err)
	}

	// Update activity with parsed metrics
	dbActivity.Duration = int(metrics.Duration.Seconds())
	dbActivity.Distance = metrics.Distance
	dbActivity.MaxHeartRate = metrics.MaxHeartRate
	dbActivity.AvgHeartRate = metrics.AvgHeartRate
	dbActivity.AvgPower = metrics.AvgPower
	dbActivity.Calories = metrics.Calories
	dbActivity.Downloaded = true
	dbActivity.Filename = filename
	dbActivity.FileType = "fit"

	// Save updated activity
	if err := s.db.UpdateActivity(dbActivity); err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}

	fmt.Printf("Successfully synced activity %d\n", activity.ActivityID)
	return nil
}

func parseTime(timeStr string) time.Time {
	// Garmin time format: "2023-08-15 12:30:45"
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return time.Now()
	}
	return t
}
