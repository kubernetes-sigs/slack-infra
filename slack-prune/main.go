/*
Copyright The Kubernetes Authors.

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

// Command slack-prune reduces clutter in the Kubernetes Slack by acting on
// inactive accounts. Inactivity is measured by workspace-wide last activity
// from team.accessLogs, since Slack exposes no per-channel activity signal.
//
// Modes:
//
//	report        walk access logs and report volume/runtime only.
//	channel-kick  remove long-inactive members from configured channels.
//	              Defaults to a dry run.
//	deactivate    deactivate long-inactive accounts workspace-wide.
//	              Defaults to a dry run.
package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

// Slack limits: team.accessLogs allows at most 1000 entries per page and 100
// pages per query window. To page further back than that, we re-anchor the
// query with the `before` parameter and start again at page 1.
const (
	maxPageSize = 1000
	maxPages    = 100
)

type options struct {
	mode        string
	configPath  string
	cutoff      time.Duration
	pageSize    int
	maxAPICalls int
	progressN   int
	dumpActive  bool

	checkpointPath  string
	checkpointEvery int
	storeLocation   string

	// channel-kick mode
	channels string
	maxKicks int

	// deactivate mode
	maxDeactivations int

	// shared by channel-kick and deactivate
	dryRun bool
	allow  string
}

func parseFlags() options {
	o := options{}
	flag.StringVar(&o.mode, "mode", "report", "what to do: report | channel-kick | deactivate")
	flag.StringVar(&o.configPath, "config", "config.json", "path to the slack auth config file")
	// Default 2 years, matching the initial threshold agreed for #kubernetes-users.
	flag.DurationVar(&o.cutoff, "cutoff", 2*365*24*time.Hour, "inactivity threshold: a user with no activity for longer than this is a prune candidate")
	flag.IntVar(&o.pageSize, "page-size", maxPageSize, "access-log entries to request per API call (max 1000)")
	flag.IntVar(&o.maxAPICalls, "max-api-calls", 5000, "safety cap on access-log API calls per invocation (0 = unlimited); with --checkpoint this pauses resumably, without it the walk bails")
	flag.IntVar(&o.progressN, "progress-every", 25, "log access-log progress every N API calls")
	flag.BoolVar(&o.dumpActive, "dump-active", false, "(report mode) print every user found active since the cutoff")
	flag.StringVar(&o.checkpointPath, "checkpoint", "", "path to a checkpoint file; enables a resumable access-log walk that survives restarts (needed to complete a full walk that may take hours)")
	flag.IntVar(&o.checkpointEvery, "checkpoint-every", 200, "save the checkpoint every N API calls")
	flag.StringVar(&o.storeLocation, "store", "", "(channel-kick, deactivate) local path or gs:// URL of the durable activity store; incremental across runs. Empty = full walk every run")

	flag.StringVar(&o.channels, "channels", "kubernetes-users", "(channel-kick mode) comma-separated channel names to prune")
	flag.IntVar(&o.maxKicks, "max-kicks", 500, "(channel-kick mode) safety cap on kicks performed per run")
	flag.IntVar(&o.maxDeactivations, "max-deactivations", 1000, "(deactivate mode) safety cap on deactivations performed per run")
	flag.BoolVar(&o.dryRun, "dry-run", true, "(channel-kick, deactivate) if true (the default), only report what would be done")
	flag.StringVar(&o.allow, "allow-users", "", "(channel-kick, deactivate) comma-separated user IDs or usernames to never act on")
	flag.Parse()
	return o
}

func main() {
	o := parseFlags()
	if o.pageSize > maxPageSize {
		log.Printf("page-size %d exceeds Slack's maximum of %d; clamping", o.pageSize, maxPageSize)
		o.pageSize = maxPageSize
	}

	config, err := slack.LoadConfig(o.configPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", o.configPath, err)
	}
	client := slack.New(config)

	switch o.mode {
	case "report":
		runReport(client, o)
	case "channel-kick":
		runChannelKick(client, config, o)
	case "deactivate":
		runDeactivate(client, o)
	default:
		log.Fatalf("unknown --mode %q (want: report | channel-kick | deactivate)", o.mode)
	}
}

// splitList parses a comma-separated flag value into a slice, trimming spaces
// and dropping empties.
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
