// internal/garmin/client.go
package garmin

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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
            Jar:     nil, // Don't use cookie jar, we'll manage cookies manually
        },
        baseURL: "https://connect.garmin.com",
        session: &Session{
            Username:  os.Getenv("GARMIN_EMAIL"),
            Password:  os.Getenv("GARMIN_PASSWORD"),
            UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
        },
    }
}

func (c *Client) Login() error {
    if c.session.Username == "" || c.session.Password == "" {
        return fmt.Errorf("GARMIN_EMAIL and GARMIN_PASSWORD environment variables required")
    }
    
    fmt.Printf("DEBUG: Attempting login for user: %s\n", c.session.Username)
    
    // Add random delay to look more human
    time.Sleep(time.Duration(rand.Intn(1500)+1000) * time.Millisecond)

    // Step 1: Get the initial login page to establish session
    loginURL := "https://connect.garmin.com/signin/"
    req, err := http.NewRequest("GET", loginURL, nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("User-Agent", c.session.UserAgent)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to get login page: %w", err)
    }
    defer resp.Body.Close()
    
    fmt.Printf("DEBUG: Initial login page status: %d\n", resp.StatusCode)
    
    // Store cookies
    c.session.Cookies = resp.Cookies()
    fmt.Printf("DEBUG: Received %d cookies from login page\n", len(c.session.Cookies))
    
    // Step 2: Submit login credentials
    loginData := url.Values{}
    loginData.Set("username", c.session.Username)
    loginData.Set("password", c.session.Password)
    loginData.Set("embed", "false")
    loginData.Set("displayNameRequired", "false")
    
    // Add another delay before POST
    time.Sleep(time.Duration(rand.Intn(1500)+1000) * time.Millisecond)

    req, err = http.NewRequest("POST", loginURL, strings.NewReader(loginData.Encode()))
    if err != nil {
        return err
    }
    
    // Add extra headers
    req.Header.Set("Accept-Encoding", "gzip, deflate, br")
    req.Header.Set("Connection", "keep-alive")
    req.Header.Set("Pragma", "no-cache")
    req.Header.Set("Cache-Control", "no-cache")
    req.Header.Set("Upgrade-Insecure-Requests", "1")
    
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("User-Agent", c.session.UserAgent)
    req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
    req.Header.Set("Accept-Language", "en-US,en;q=0.9")
    req.Header.Set("Origin", "https://sso.garmin.com")
    req.Header.Set("Referer", loginURL)
    req.Header.Set("X-Requested-With", "XMLHttpRequest")
    
    // Add existing cookies
    for _, cookie := range c.session.Cookies {
        req.AddCookie(cookie)
    }
    
    resp, err = c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to submit login: %w", err)
    }
    defer resp.Body.Close()
    
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }
    
    fmt.Printf("DEBUG: Login response status: %d\n", resp.StatusCode)
    fmt.Printf("DEBUG: Login response body: %s\n", string(bodyBytes))
    
    // Update cookies with login response
    for _, cookie := range resp.Cookies() {
        c.session.Cookies = append(c.session.Cookies, cookie)
    }
    
    // Check for successful login indicators
    bodyStr := string(bodyBytes)
    if strings.Contains(bodyStr, "error") || strings.Contains(bodyStr, "invalid") {
        return fmt.Errorf("login failed: %s", bodyStr)
    }
    
    // Step 3: Get the Garmin Connect session
    connectURL := "https://connect.garmin.com/modern/"
    req, err = http.NewRequest("GET", connectURL, nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("User-Agent", c.session.UserAgent)
    
    // Add all cookies
    for _, cookie := range c.session.Cookies {
        req.AddCookie(cookie)
    }
    
    resp, err = c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to access Garmin Connect: %w", err)
    }
    defer resp.Body.Close()
    
    fmt.Printf("DEBUG: Garmin Connect access status: %d\n", resp.StatusCode)
    
    // Update cookies again
    for _, cookie := range resp.Cookies() {
        c.session.Cookies = append(c.session.Cookies, cookie)
    }
    
    fmt.Printf("DEBUG: Total cookies after login: %d\n", len(c.session.Cookies))
    
    if resp.StatusCode == http.StatusOK {
        c.session.Authenticated = true
        fmt.Println("DEBUG: Login successful!")
        return nil
    }
    
    return fmt.Errorf("login failed with status: %d", resp.StatusCode)
}

func (c *Client) GetActivities(start, limit int) ([]GarminActivity, error) {
	if !c.session.Authenticated {
		if err := c.Login(); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("%s/modern/proxy/activity-service/activities/search/activities?start=%d&limit=%d",
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

	// Log cookies being sent
	fmt.Println("DEBUG: Cookies being sent:")
	for _, cookie := range req.Cookies() {
		fmt.Printf("  %s: %s (Expires: %s)\n", 
			cookie.Name, 
			cookie.Value[:min(3, len(cookie.Value))] + "***", 
			cookie.Expires.Format(time.RFC1123))
		
		// Check if cookie is expired
		if !cookie.Expires.IsZero() && cookie.Expires.Before(time.Now()) {
			fmt.Printf("WARNING: Cookie %s expired at %s\n", 
				cookie.Name, 
				cookie.Expires.Format(time.RFC1123))
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: HTTP Status: %d\n", resp.StatusCode)
	fmt.Printf("DEBUG: Response Headers: %v\n", resp.Header)
	
	// If we get empty response but 200 status, check session expiration
	if resp.StatusCode == http.StatusOK && resp.ContentLength == 2 {
		fmt.Println("WARNING: Empty API response with 200 status - checking session validity")
		c.session.Authenticated = false
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log full response for debugging
	fmt.Printf("DEBUG: Full API Response (%d bytes):\n", len(bodyBytes))
	fmt.Println(string(bodyBytes))

	// Check for empty response
	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("API returned empty response")
	}

	// Special case for empty object
	if string(bodyBytes) == "{}" {
		fmt.Println("DEBUG: API returned empty object")
		return nil, fmt.Errorf("API returned empty object")
	}

	// Try flexible parsing
	activities, err := parseActivityResponse(bodyBytes)
	if err != nil {
		fmt.Printf("DEBUG: Failed to parse activities: %v\n", err)
		return nil, err
	}

	fmt.Printf("DEBUG: Successfully parsed %d activities\n", len(activities))
	
	// Rate limiting
	time.Sleep(2 * time.Second)
	
	return activities, nil
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseActivityResponse handles different API response formats
func parseActivityResponse(bodyBytes []byte) ([]GarminActivity, error) {
	// Try standard ActivityList format
	type ActivityListResponse struct {
		ActivityList []GarminActivity `json:"activityList"`
	}
	var listResponse ActivityListResponse
	if err := json.Unmarshal(bodyBytes, &listResponse); err == nil && len(listResponse.ActivityList) > 0 {
		return listResponse.ActivityList, nil
	}

	// Try direct array format
	var directResponse []GarminActivity
	if err := json.Unmarshal(bodyBytes, &directResponse); err == nil && len(directResponse) > 0 {
		return directResponse, nil
	}

	// Try generic map-based format
	var genericResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &genericResponse); err == nil {
		// Check if we have an "activityList" key
		if activityList, ok := genericResponse["activityList"].([]interface{}); ok {
			return convertInterfaceSlice(activityList)
		}
		// Check if we have a "results" key
		if results, ok := genericResponse["results"].([]interface{}); ok {
			return convertInterfaceSlice(results)
		}
		// Check if we have an "activities" key
		if activities, ok := genericResponse["activities"].([]interface{}); ok {
			return convertInterfaceSlice(activities)
		}
	}

	// Failed to parse
	return nil, fmt.Errorf("unable to parse API response")
}

// convertInterfaceSlice converts []interface{} to []GarminActivity
func convertInterfaceSlice(items []interface{}) ([]GarminActivity, error) {
	var activities []GarminActivity
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Convert map to JSON then to GarminActivity
		jsonData, err := json.Marshal(itemMap)
		if err != nil {
			return nil, err
		}

		var activity GarminActivity
		if err := json.Unmarshal(jsonData, &activity); err != nil {
			return nil, err
		}

		activities = append(activities, activity)
	}
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
