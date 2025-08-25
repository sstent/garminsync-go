package parser

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/tormoder/fit"
	"github.com/sstent/garminsync-go/internal/models"
)

// Parser handles FIT file parsing
type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseFile(filename string) (*models.ActivityMetrics, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return p.ParseData(data)
}

func (p *Parser) ParseData(data []byte) (*models.ActivityMetrics, error) {
	fitFile, err := fit.Decode(bytes.NewReader(data))
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
	metrics := &models.ActivityMetrics{
		StartTime:     session.StartTime,
		Duration:      time.Duration(session.TotalTimerTime) * time.Second,
		Distance:      float64(session.TotalDistance),
		AvgHeartRate:  int(session.AvgHeartRate),
		MaxHeartRate:  int(session.MaxHeartRate),
		AvgPower:      int(session.AvgPower),
		Calories:      int(session.TotalCalories),
		ElevationGain: float64(session.TotalAscent),
		ElevationLoss: float64(session.TotalDescent),
		Steps:         0, // FIT sessions don't include steps
	}

	return metrics, nil
}
