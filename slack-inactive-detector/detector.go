/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

// InactiveDetector detects inactive users in Slack workspace or channels
type InactiveDetector struct {
	client        *slack.Client
	inactiveYears int
	channelID     string
	dryRun        bool
}

// UserActivity represents user activity information
type UserActivity struct {
	User         slack.User
	LastActivity time.Time
	IsInactive   bool
}

// UsersListResponse represents the response from users.list API
type UsersListResponse struct {
	OK               bool         `json:"ok"`
	Members          []slack.User `json:"members"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

// UserPresenceResponse represents the response from users.getPresence API
type UserPresenceResponse struct {
	OK              bool      `json:"ok"`
	Presence        string    `json:"presence"`
	Online          bool      `json:"online"`
	AutoAway        bool      `json:"auto_away"`
	ManualAway      bool      `json:"manual_away"`
	ConnectionCount int       `json:"connection_count"`
	LastActivity    time.Time `json:"last_activity"`
}

// DetectInactiveUsers finds and reports inactive users
func (d *InactiveDetector) DetectInactiveUsers() error {
	log.Printf("Detecting inactive users (threshold: %d years)", d.inactiveYears)

	// Get all users
	users, err := d.getAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %v", err)
	}

	log.Printf("Found %d total users", len(users))

	// Calculate cutoff time for inactivity
	cutoffTime := time.Now().AddDate(-d.inactiveYears, 0, 0)
	log.Printf("Considering users inactive if last activity before: %s", cutoffTime.Format("2006-01-02"))

	var inactiveUsers []UserActivity

	// Check each user's activity
	for _, user := range users {
		// Skip bots and deleted users
		if user.IsBot || user.Deleted {
			continue
		}

		activity, err := d.getUserActivity(user)
		if err != nil {
			log.Printf("Warning: could not get activity for user %s (%s): %v", user.Name, user.ID, err)
			continue
		}

		if activity.LastActivity.Before(cutoffTime) {
			activity.IsInactive = true
			inactiveUsers = append(inactiveUsers, activity)
		}
	}

	// Report results
	d.reportResults(inactiveUsers, cutoffTime)

	return nil
}

// getAllUsers retrieves all users from the Slack workspace
func (d *InactiveDetector) getAllUsers() ([]slack.User, error) {
	var allUsers []slack.User
	cursor := ""

	for {
		args := map[string]string{
			"limit": "200",
		}
		if cursor != "" {
			args["cursor"] = cursor
		}

		var response UsersListResponse
		err := d.client.CallOldMethod("users.list", args, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to call users.list: %v", err)
		}

		if !response.OK {
			return nil, fmt.Errorf("users.list returned ok=false")
		}

		allUsers = append(allUsers, response.Members...)

		if response.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = response.ResponseMetadata.NextCursor
	}

	return allUsers, nil
}

// getUserActivity gets the last activity time for a user
func (d *InactiveDetector) getUserActivity(user slack.User) (UserActivity, error) {
	activity := UserActivity{
		User:         user,
		LastActivity: time.Unix(0, 0), // Default to epoch if no activity found
	}

	// Try to get user presence information
	args := map[string]string{
		"user": user.ID,
	}

	var presence UserPresenceResponse
	err := d.client.CallOldMethod("users.getPresence", args, &presence)
	if err != nil {
		// If we can't get presence, we'll use a fallback approach
		log.Printf("Could not get presence for user %s: %v", user.Name, err)

		// Fallback: use a very old date to mark as potentially inactive
		// In a real implementation, you might want to check conversation history
		activity.LastActivity = time.Unix(0, 0)
		return activity, nil
	}

	if presence.OK && !presence.LastActivity.IsZero() {
		activity.LastActivity = presence.LastActivity
	} else {
		// If no last activity in presence, try to estimate from user creation or other means
		// For now, we'll mark as very old
		activity.LastActivity = time.Unix(0, 0)
	}

	return activity, nil
}

// reportResults prints and optionally acts on the inactive users found
func (d *InactiveDetector) reportResults(inactiveUsers []UserActivity, cutoffTime time.Time) {
	if len(inactiveUsers) == 0 {
		log.Printf("✅ No inactive users found!")
		return
	}

	log.Printf("🔍 Found %d inactive users (last activity before %s):",
		len(inactiveUsers), cutoffTime.Format("2006-01-02"))

	for _, activity := range inactiveUsers {
		lastActivityStr := "unknown"
		if !activity.LastActivity.IsZero() && activity.LastActivity.Unix() > 0 {
			lastActivityStr = activity.LastActivity.Format("2006-01-02")
		}

		log.Printf("  - %s (%s) - Last activity: %s",
			activity.User.Name, activity.User.Profile.RealName, lastActivityStr)
	}

	if d.dryRun {
		log.Printf("\n🔬 DRY RUN: No actions taken. Use -dry-run=false to take action.")
	} else {
		log.Printf("\n⚠️  To implement: Add actual remediation actions here (deactivate, send warnings, etc.)")
		// TODO: Implement actual actions like deactivating users or sending warnings
	}
}
