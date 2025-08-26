package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sst/garmin-sync-go/internal/garmin"
)

func main() {
	// Set environment variables for testing
	os.Setenv("GARMIN_EMAIL", "your_email@example.com")
	os.Setenv("GARMIN_PASSWORD", "your_password")

	client := garmin.NewClient()

	start := time.Now()
	fmt.Println("Starting login test...")
	
	err := client.Login()
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Login successful! Duration: %v\n", time.Since(start))
	
	if client.IsAuthenticated() {
		fmt.Println("Session authenticated:", client.IsAuthenticated())
		fmt.Println("Number of cookies:", len(client.GetCookies()))
	} else {
		fmt.Println("Session not authenticated after successful login")
		os.Exit(1)
	}
}
