// internal/garmin/client.go
package garmin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
    httpClient *http.Client
    baseURL    string
}

type GarminActivity struct {
    ActivityID       int                    `json:"activityId"`
    ActivityName     string                 `json:"activityName"`
    StartTimeLocal   string                 `json:"startTimeLocal"`
    ActivityType     map[string]interface{} `json:"activityType"`
    Distance         float64                `json:"distance"`
    Duration         float64                `json:"duration"`
    MaxHR            float64                `json:"maxHR"`
    AvgHR            float64                `json:"avgHR"`
    AvgPower         float64                `json:"avgPower"`
    Calories         float64                `json:"calories"`
    StartLatitude    float64                `json:"startLatitude"`
    StartLongitude   float64                `json:"startLongitude"`
    Steps            float64                `json:"steps"`
    ElevationGain    float64                `json:"elevationGain"`
    ElevationLoss    float64                `json:"elevationLoss"`
    AvgTemperature   float64                `json:"avgTemperature"`
    MinTemperature   float64                `json:"minTemperature"`
    MaxTemperature   float64                `json:"maxTemperature"`
}

// NewClient creates a new Garmin API client
func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        baseURL: "http://garmin-api:8081",
    }
}

// GetStats retrieves user statistics for a specific date via the Python API service
func (c *Client) GetStats(date string) (map[string]interface{}, error) {
    // Construct request URL
    url := fmt.Sprintf("%s/stats?date=%s", c.baseURL, url.QueryEscape(date))
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, body)
    }
    
    var stats map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
        return nil, err
    }
    
    return stats, nil
}

// GetActivities retrieves activities from the Python API wrapper
func (c *Client) GetActivities(start, limit int) ([]GarminActivity, error) {
    url := fmt.Sprintf("%s/activities?start=%d&limit=%d", c.baseURL, start, limit)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, body)
    }
    
    var activities []GarminActivity
    if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
        return nil, err
    }
    
    return activities, nil
}

// Helper function removed - no longer needed

// DownloadActivity downloads an activity from Garmin Connect (stub implementation)
func (c *Client) DownloadActivity(activityID int, format string) ([]byte, error) {
	return nil, fmt.Errorf("DownloadActivity not implemented - use Python API")
}

// GetActivityDetails retrieves details for a specific activity from the Python API wrapper
func (c *Client) GetActivityDetails(activityID int) (*GarminActivity, error) {
    url := fmt.Sprintf("%s/activities/%d", c.baseURL, activityID)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, body)
    }
    
    var activity GarminActivity
    if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
        return nil, err
    }
    
    return &activity, nil
}
