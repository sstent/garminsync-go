// internal/parser/activity.go
package parser

import (
    "encoding/xml"
    "fmt"
    "math"
    "os"
    "time"
)

type ActivityMetrics struct {
    ActivityType string
    Duration     int     // seconds
    Distance     float64 // meters
    MaxHR        int
    AvgHR        int
    AvgPower     float64
    Calories     int
    StartTime    time.Time
}

type Parser interface {
    ParseFile(filepath string) (*ActivityMetrics, error)
}

func NewParser(fileType FileType) Parser {
    switch fileType {
    case FileTypeFIT:
        return &FITParser{}
    case FileTypeTCX:
        return &TCXParser{}
    case FileTypeGPX:
        return &GPXParser{}
    default:
        return nil
    }
}

// TCX Parser Implementation
type TCXParser struct{}

type TCXTrainingCenterDatabase struct {
    Activities TCXActivities `xml:"Activities"`
}

type TCXActivities struct {
    Activity []TCXActivity `xml:"Activity"`
}

type TCXActivity struct {
    Sport string    `xml:"Sport,attr"`
    Laps  []TCXLap  `xml:"Lap"`
}

type TCXLap struct {
    StartTime        string  `xml:"StartTime,attr"`
    TotalTimeSeconds float64 `xml:"TotalTimeSeconds"`
    DistanceMeters   float64 `xml:"DistanceMeters"`
    Calories         int     `xml:"Calories"`
    MaximumSpeed     float64 `xml:"MaximumSpeed"`
    AverageHeartRate TCXHeartRate `xml:"AverageHeartRateBpm"`
    MaximumHeartRate TCXHeartRate `xml:"MaximumHeartRateBpm"`
    Track            TCXTrack     `xml:"Track"`
}

type TCXHeartRate struct {
    Value int `xml:"Value"`
}

type TCXTrack struct {
    Trackpoints []TCXTrackpoint `xml:"Trackpoint"`
}

type TCXTrackpoint struct {
    Time         string       `xml:"Time"`
    HeartRateBpm TCXHeartRate `xml:"HeartRateBpm"`
}

func (p *TCXParser) ParseFile(filepath string) (*ActivityMetrics, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var tcx TCXTrainingCenterDatabase
    decoder := xml.NewDecoder(file)
    if err := decoder.Decode(&tcx); err != nil {
        return nil, err
    }
    
    if len(tcx.Activities.Activity) == 0 || len(tcx.Activities.Activity[0].Laps) == 0 {
        return nil, fmt.Errorf("no activity data found")
    }
    
    activity := tcx.Activities.Activity[0]
    firstLap := activity.Laps[0]
    
    metrics := &ActivityMetrics{
        ActivityType: mapTCXSportType(activity.Sport),
    }
    
    // Parse start time
    if startTime, err := time.Parse(time.RFC3339, firstLap.StartTime); err == nil {
        metrics.StartTime = startTime
    }
    
    // Aggregate data from all laps
    var totalDuration, totalDistance float64
    var maxHR, totalCalories int
    var hrValues []int
    
    for _, lap := range activity.Laps {
        totalDuration += lap.TotalTimeSeconds
        totalDistance += lap.DistanceMeters
        totalCalories += lap.Calories
        
        if lap.MaximumHeartRate.Value > maxHR {
            maxHR = lap.MaximumHeartRate.Value
        }
        
        if lap.AverageHeartRate.Value > 0 {
            hrValues = append(hrValues, lap.AverageHeartRate.Value)
        }
        
        // Collect HR data from trackpoints
        for _, tp := range lap.Track.Trackpoints {
            if tp.HeartRateBpm.Value > 0 {
                hrValues = append(hrValues, tp.HeartRateBpm.Value)
            }
        }
    }
    
    metrics.Duration = int(totalDuration)
    metrics.Distance = totalDistance
    metrics.MaxHR = maxHR
    metrics.Calories = totalCalories
    
    // Calculate average HR
    if len(hrValues) > 0 {
        sum := 0
        for _, hr := range hrValues {
            sum += hr
        }
        metrics.AvgHR = sum / len(hrValues)
    }
    
    return metrics, nil
}

func mapTCXSportType(sport string) string {
    switch sport {
    case "Running":
        return "running"
    case "Biking":
        return "cycling"
    case "Swimming":
        return "swimming"
    default:
        return "other"
    }
}

// GPX Parser Implementation
type GPXParser struct{}

type GPX struct {
    Tracks []GPXTrack `xml:"trk"`
}

type GPXTrack struct {
    Name     string       `xml:"name"`
    Segments []GPXSegment `xml:"trkseg"`
}

type GPXSegment struct {
    Points []GPXPoint `xml:"trkpt"`
}

type GPXPoint struct {
    Lat       float64   `xml:"lat,attr"`
    Lon       float64   `xml:"lon,attr"`
    Elevation float64   `xml:"ele"`
    Time      string    `xml:"time"`
    HR        int       `xml:"extensions>TrackPointExtension>hr"`
}

func (p *GPXParser) ParseFile(filepath string) (*ActivityMetrics, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var gpx GPX
    decoder := xml.NewDecoder(file)
    if err := decoder.Decode(&gpx); err != nil {
        return nil, err
    }
    
    if len(gpx.Tracks) == 0 || len(gpx.Tracks[0].Segments) == 0 {
        return nil, fmt.Errorf("no track data found")
    }
    
    metrics := &ActivityMetrics{
        ActivityType: "other", // GPX doesn't specify activity type
    }
    
    var allPoints []GPXPoint
    for _, track := range gpx.Tracks {
        for _, segment := range track.Segments {
            allPoints = append(allPoints, segment.Points...)
        }
    }
    
    if len(allPoints) == 0 {
        return nil, fmt.Errorf("no track points found")
    }
    
    // Calculate metrics from points
    var startTime, endTime time.Time
    var totalDistance float64
    var hrValues []int
    
    for i, point := range allPoints {
        // Parse time
        if point.Time != "" {
            if t, err := time.Parse(time.RFC3339, point.Time); err == nil {
                if i == 0 {
                    startTime = t
                    metrics.StartTime = t
                }
                endTime = t
            }
        }
        
        // Calculate distance between consecutive points
        if i > 0 {
            prevPoint := allPoints[i-1]
            distance := calculateDistance(prevPoint.Lat, prevPoint.Lon, point.Lat, point.Lon)
            totalDistance += distance
        }
        
        // Collect heart rate data
        if point.HR > 0 {
            hrValues = append(hrValues, point.HR)
        }
    }
    
    // Calculate duration
    if !startTime.IsZero() && !endTime.IsZero() {
        metrics.Duration = int(endTime.Sub(startTime).Seconds())
    }
    
    metrics.Distance = totalDistance
    
    // Calculate heart rate metrics
    if len(hrValues) > 0 {
        sum := 0
        maxHR := 0
        for _, hr := range hrValues {
            sum += hr
            if hr > maxHR {
                maxHR = hr
            }
        }
        metrics.AvgHR = sum / len(hrValues)
        metrics.MaxHR = maxHR
    }
    
    return metrics, nil
}

// Haversine formula for distance calculation
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
    const earthRadius = 6371000 // Earth's radius in meters
    
    dLat := (lat2 - lat1) * math.Pi / 180
    dLon := (lon2 - lon1) * math.Pi / 180
    
    lat1Rad := lat1 * math.Pi / 180
    lat2Rad := lat2 * math.Pi / 180
    
    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
    
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    
    return earthRadius * c
}

// FIT Parser Implementation (simplified - would use FIT SDK in real implementation)
type FITParser struct{}

func (p *FITParser) ParseFile(filepath string) (*ActivityMetrics, error) {
    // For now, return basic metrics - in real implementation, would use FIT SDK
    // This is a placeholder that reads basic file info
    
    file, err := os.Open(filepath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    // Read FIT header to verify it's a valid FIT file
    header := make([]byte, 14)
    _, err = file.Read(header)
    if err != nil {
        return nil, err
    }
    
    // Verify FIT signature
    if !bytes.Equal(header[8:12], []byte(".FIT")) {
        return nil, fmt.Errorf("invalid FIT file signature")
    }
    
    // For now, return empty metrics - real implementation would parse FIT records
    return &ActivityMetrics{
        ActivityType: "other",
        // Additional parsing would happen here using FIT SDK
    }, nil
}
