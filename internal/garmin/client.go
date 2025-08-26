package garmin

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	retries    int // Number of retries for failed requests
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "http://garmin-api:8081",
		retries: 3, // Default to 3 retries
	}
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

func (c *Client) GetStats(date string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/stats?date=%s", c.baseURL, url.QueryEscape(date))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	var bodyBytes []byte
	reqErr := error(nil)

	for i := 0; i <= c.retries; i++ {
		resp, reqErr = c.httpClient.Do(req)
		if reqErr != nil || (resp != nil && resp.StatusCode >= 500) {
			if i < c.retries {
				backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
				log.Printf("Request failed (attempt %d/%d), retrying in %v: %v", i+1, c.retries, backoff, reqErr)
				time.Sleep(backoff)
				continue
			}
		}
		break
	}

	if reqErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.retries, reqErr)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, bodyBytes)
	}

	var stats map[string]interface{}
	if jsonErr := json.Unmarshal(bodyBytes, &stats); jsonErr != nil {
		return nil, jsonErr
	}

	return stats, nil
}

func (c *Client) GetActivities(start, limit int) ([]GarminActivity, error) {
	url := fmt.Sprintf("%s/activities?start=%d&limit=%d", c.baseURL, start, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	var bodyBytes []byte
	reqErr := error(nil)

	for i := 0; i <= c.retries; i++ {
		resp, reqErr = c.httpClient.Do(req)
		if reqErr != nil || (resp != nil && resp.StatusCode >= 500) {
			if i < c.retries {
				backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
				log.Printf("Request failed (attempt %d/%d), retrying in %v: %v", i+1, c.retries, backoff, reqErr)
				time.Sleep(backoff)
				continue
			}
		}
		break
	}

	if reqErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.retries, reqErr)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, bodyBytes)
	}

	var activities []GarminActivity
	if jsonErr := json.Unmarshal(bodyBytes, &activities); jsonErr != nil {
		return nil, jsonErr
	}

	return activities, nil
}

func (c *Client) DownloadActivity(activityID int, format string) ([]byte, error) {
	url := fmt.Sprintf("%s/activities/%d/download?format=%s", c.baseURL, activityID, format)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	reqErr := error(nil)

	for i := 0; i <= c.retries; i++ {
		resp, reqErr = c.httpClient.Do(req)
		if reqErr != nil || (resp != nil && resp.StatusCode >= 500) {
			if i < c.retries {
				backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
				log.Printf("Download failed (attempt %d/%d), retrying in %v: %v", i+1, c.retries, backoff, reqErr)
				time.Sleep(backoff)
				continue
			}
		}
		break
	}

	if reqErr != nil {
		return nil, fmt.Errorf("download failed after %d retries: %w", c.retries, reqErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) GetActivityDetails(activityID int) (*GarminActivity, error) {
	url := fmt.Sprintf("%s/activities/%d", c.baseURL, activityID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	var bodyBytes []byte
	reqErr := error(nil)

	for i := 0; i <= c.retries; i++ {
		resp, reqErr = c.httpClient.Do(req)
		if reqErr != nil || (resp != nil && resp.StatusCode >= 500) {
			if i < c.retries {
				backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
				log.Printf("Request failed (attempt %d/%d), retrying in %v: %v", i+1, c.retries, backoff, reqErr)
				time.Sleep(backoff)
				continue
			}
		}
		break
	}

	if reqErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.retries, reqErr)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, bodyBytes)
	}

	var activity GarminActivity
	if jsonErr := json.Unmarshal(bodyBytes, &activity); jsonErr != nil {
		return nil, jsonErr
	}

	return &activity, nil
}
