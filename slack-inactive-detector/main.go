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
	"flag"
	"log"
	"os"
	"strconv"

	"sigs.k8s.io/slack-infra/slack"
)

type options struct {
	configPath    string
	inactiveYears int
	channelID     string
	dryRun        bool
}

func parseFlags() options {
	o := options{}
	flag.StringVar(&o.configPath, "config-path", "config.json", "Path to a file containing the slack config")
	flag.StringVar(&o.channelID, "channel", "", "Specific channel ID to check for inactive users (optional, checks workspace if not provided)")
	flag.BoolVar(&o.dryRun, "dry-run", true, "Only report inactive users, don't take any action")
	flag.Parse()

	// Get inactivity period from environment variable, default to 1 year
	inactiveYearsStr := os.Getenv("INACTIVE_YEARS")
	if inactiveYearsStr == "" {
		o.inactiveYears = 1
	} else {
		years, err := strconv.Atoi(inactiveYearsStr)
		if err != nil || years < 1 {
			log.Fatalf("Invalid INACTIVE_YEARS value: %s. Must be a positive integer.", inactiveYearsStr)
		}
		o.inactiveYears = years
	}

	return o
}

func main() {
	o := parseFlags()

	log.Printf("Starting slack-inactive-detector with %d year(s) inactivity threshold", o.inactiveYears)

	c, err := slack.LoadConfig(o.configPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", o.configPath, err)
	}

	client := slack.New(c)
	detector := &InactiveDetector{
		client:        client,
		inactiveYears: o.inactiveYears,
		channelID:     o.channelID,
		dryRun:        o.dryRun,
	}

	if err := detector.DetectInactiveUsers(); err != nil {
		log.Fatalf("Failed to detect inactive users: %v", err)
	}
}
