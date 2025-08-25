package parser

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tormoder/fit"
	"github.com/sstent/garminsync-go/internal/models"
)

type FITParser struct{}

func NewFITParser() *FITParser {
	return &FITParser{}
}

func (p *FITParser) ParseFile(filename string) (*models.ActivityMetrics, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return p.ParseData(data)
}

func (p *FITParser) ParseData(data []byte) (*models.ActivityMetrics, error) {
	fitFile, err := fit.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode FIT file: %w", err)
	}

	activity, err := fitFile.Activity()
	if err != nil {
		return nil, fmt.Errorf("failed to get activity from FIT: %w", err)
	}

	if len(activity.Sessions) == 0 {
		return nil, fmt.Errorf("no sessions found in FIT file")
	}

	session := activity.Sessions[0]
	metrics := &models.ActivityMetrics{}

	// Basic activity metrics
	metrics.StartTime = session.StartTime
	metrics.Duration = time.Duration(session.TotalTimerTime) * time.Second
	metrics.Distance = session.TotalDistance

	// Heart rate
	if session.AvgHeartRate != nil {
		metrics.AvgHeartRate = int(*session.AvgHeartRate)
	}
	if session.MaxHeartRate != nil {
		metrics.MaxHeartRate = int(*session.MaxHeartRate)
	}

	// Power
	if session.AvgPower != nil {
		metrics.AvgPower = int(*session.AvgPower)
	}

	// Calories
	if session.TotalCalories != nil {
		metrics.Calories = int(*session.TotalCalories)
	}

	// Elevation
	if session.TotalAscent != nil {
		metrics.ElevationGain = *session.TotalAscent
	}
	if session.TotalDescent != nil {
		metrics.ElevationLoss = *session.TotalDescent
	}

	// Steps
	if session.Steps != nil {
		metrics.Steps = int(*session.Steps)
	}

	// Temperature - FIT typically doesn't store temp in session summary
	// We'll leave temperature fields as 0 for FIT files

	return metrics, nil
}
