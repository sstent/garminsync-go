package parser

import (
	"encoding/xml"
	"math"
	"time"

	"github.com/sstent/garminsync-go/internal/parser"
)

// GPX represents the root element of a GPX file
type GPX struct {
	XMLName xml.Name `xml:"gpx"`
	Trk     Trk      `xml:"trk"`
}

// Trk represents a track in a GPX file
type Trk struct {
	Name   string  `xml:"name"`
	TrkSeg []TrkSeg `xml:"trkseg"`
}

// TrkSeg represents a track segment in a GPX file
type TrkSeg struct {
	TrkPt []TrkPt `xml:"trkpt"`
}

// TrkPt represents a track point in a GPX file
type TrkPt struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Ele  float64 `xml:"ele"`
	Time string  `xml:"time"`
}

// GPXParser implements the Parser interface for GPX files
type GPXParser struct{}

func (p *GPXParser) Parse(data []byte) (*activity.Activity, error) {
	var gpx GPX
	if err := xml.Unmarshal(data, &gpx); err != nil {
		return nil, err
	}

	if len(gpx.Trk.TrkSeg) == 0 || len(gpx.Trk.TrkSeg[0].TrkPt) == 0 {
		return nil, ErrNoTrackData
	}

	// Process track points
	points := gpx.Trk.TrkSeg[0].TrkPt
	startTime, _ := time.Parse(time.RFC3339, points[0].Time)
	endTime, _ := time.Parse(time.RFC3339, points[len(points)-1].Time)
	
	activity := &activity.Activity{
		ActivityType:  "hiking",
		StartTime:     startTime,
		Duration:      int(endTime.Sub(startTime).Seconds()),
		StartLatitude: points[0].Lat,
		StartLongitude: points[0].Lon,
	}

	// Calculate distance and elevation
	var totalDistance, elevationGain float64
	prev := points[0]
	
	for i := 1; i < len(points); i++ {
		curr := points[i]
		totalDistance += haversine(prev.Lat, prev.Lon, curr.Lat, curr.Lon)
		
		if curr.Ele > prev.Ele {
			elevationGain += curr.Ele - prev.Ele
		}
		prev = curr
	}

	activity.Distance = totalDistance
	activity.ElevationGain = elevationGain

	return activity, nil
}

// haversine calculates the distance between two points on Earth
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth radius in meters
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*
			math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func init() {
	RegisterParser(".gpx", &GPXParser{})
}
