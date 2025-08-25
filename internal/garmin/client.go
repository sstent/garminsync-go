// internal/garmin/client.go
package garmin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
    httpClient *http.Client
    baseURL    string
    session    *Session
}

type Session struct {
    Username    string
    Password    string
    Cookies     []*http.Cookie
    UserAgent   string
    Authenticated bool
}

type GarminActivity struct {
    ActivityID       int                    `json:"activityId"`
    ActivityName     string                 `json:"activityName"`
    StartTimeLocal   string                 `json:"startTimeLocal"`
    ActivityType     map[string]interface{} `json:"activityType"`
    Distance         float64                `json:"distance"`
    Duration         float64                `json:"duration"`
    MaxHR            int                    `json:"maxHR"`
    AvgHR            int                    `json:"avgHR"`
    AvgPower         float64                `json:"avgPower"`
    Calories         int                    `json:"calories"`
    StartLatitude    float64                `json:"startLatitude"`
    StartLongitude   float64                `json:"startLongitude"`
    Steps            int                    `json:"steps"`
    ElevationGain    float64                `json:"elevationGain"`
    ElevationLoss    float64                `json:"elevationLoss"`
    AvgTemperature   float64                `json:"avgTemperature"`
    MinTemperature   float64                `json:"minTemperature"`
    MaxTemperature   float64                `json:"maxTemperature"`
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        baseURL: "https://connect.garmin.com",
        session: &Session{
            Username:  os.Getenv("GARMIN_EMAIL"),
            Password:  os.Getenv("GARMIN_PASSWORD"),
            UserAgent: "GarminSync/1.0",
        },
    }
}

func (c *Client) Login() error {
    if c.session.Username == "" || c.session.Password == "" {
        return fmt.Errorf("GARMIN_EMAIL and GARMIN_PASSWORD environment variables required")
    }
    
    // Step 1: Get login form
    loginURL := c.baseURL + "/signin"
    req, err := http.NewRequest("GET", loginURL, nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("User-Agent", c.session.UserAgent)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Extract cookies
    c.session.Cookies = resp.Cookies()
    
    // Step 2: Submit login credentials
    loginData := url.Values{}
    loginData.Set("username", c.session.Username)
    loginData.Set("password", c.session.Password)
    loginData.Set("embed", "true")
    
    req, err = http.NewRequest("POST", loginURL, strings.NewReader(loginData.Encode()))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("User-Agent", c.session.UserAgent)
    
    // Add cookies
    for _, cookie := range c.session.Cookies {
        req.AddCookie(cookie)
    }
    
    resp, err = c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Check if login was successful
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("login failed with status: %d", resp.StatusCode)
    }
    
    // Update cookies
    for _, cookie := range resp.Cookies() {
        c.session.Cookies = append(c.session.Cookies, cookie)
    }
    
    c.session.Authenticated = true
    return nil
}

func (c *Client) GetActivities(start, limit int) ([]GarminActivity, error) {
    if !c.session.Authenticated {
        if err := c.Login(); err != nil {
            return nil, err
        }
    }
    
    url := fmt.Sprintf("%s/modern/proxy/activitylist-service/activities/search/activities?start=%d&limit=%d",
        c.baseURL, start, limit)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("User-Agent", c.session.UserAgent)
    req.Header.Set("Accept", "application/json")
    
    // Add cookies
    for _, cookie := range c.session.Cookies {
        req.AddCookie(cookie)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to get activities: status %d", resp.StatusCode)
    }
    
    var activities []GarminActivity
    if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
        return nil, err
    }
    
    // Rate limiting
    time.Sleep(2 * time.Second)
    
    return activities, nil
}

func (c *Client) DownloadActivity(activityID int, format string) ([]byte, error) {
    if !c.session.Authenticated {
        if err := c.Login(); err != nil {
            return nil, err
        }
    }
    
    // Default to FIT format
    if format == "" {
        format = "fit"
    }
    
    url := fmt.Sprintf("%s/modern/proxy/download-service/export/%s/activity/%d",
        c.baseURL, format, activityID)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("User-Agent", c.session.UserAgent)
    
    // Add cookies
    for _, cookie := range c.session.Cookies {
        req.AddCookie(cookie)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to download activity %d: status %d", activityID, resp.StatusCode)
    }
    
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    // Rate limiting
    time.Sleep(2 * time.Second)
    
    return data, nil
}

func (c *Client) GetActivityDetails(activityID int) (*GarminActivity, error) {
    if !c.session.Authenticated {
        if err := c.Login(); err != nil {
            return nil, err
        }
    }
    
    url := fmt.Sprintf("%s/modern/proxy/activity-service/activity/%d",
        c.baseURL, activityID)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("User-Agent", c.session.UserAgent)
    req.Header.Set("Accept", "application/json")
    
    // Add cookies
    for _, cookie := range c.session.Cookies {
        req.AddCookie(cookie)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to get activity details: status %d", resp.StatusCode)
    }
    
    var activity GarminActivity
    if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
        return nil, err
    }

    // Extract activity type from map if possible
    if typeKey, ok := activity.ActivityType["typeKey"].(string); ok {
        activity.ActivityType = map[string]interface{}{"typeKey": typeKey}
    } else {
        // Default to empty map if typeKey not found
        activity.ActivityType = map[string]interface{}{}
    }
    
    // Rate limiting
    time.Sleep(2 * time.Second)
    
    return &activity, nil
}
