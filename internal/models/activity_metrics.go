package models

import "time"

// ActivityMetrics contains all metrics extracted from activity files
type ActivityMetrics struct {
	ActivityType   string
	StartTime      time.Time
	Duration       time.Duration
	Distance       float64 // in meters
	MaxHeartRate   int
	AvgHeartRate   int
	AvgPower       int
	Calories       int
	Steps          int
	ElevationGain  float64 // in meters
	ElevationLoss  float64 // in meters
	MinTemperature float64 // in °C
	MaxTemperature float64 // in °C
	AvgTemperature float64 // in °C
}
