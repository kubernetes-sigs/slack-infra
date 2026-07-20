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

package main

import (
	"log"
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

// runChannelKick removes long-inactive members from the configured channels. It
// defaults to a dry run; pass --dry-run=false to actually kick.
func runChannelKick(client *slack.Client, config slack.Config, o options) {
	channelNames := splitList(o.channels)
	if len(channelNames) == 0 {
		log.Fatalf("channel-kick mode requires at least one channel in --channels")
	}

	cutoffTS := time.Now().Add(-o.cutoff).Unix()
	log.Printf("Building activity map (cutoff %s ago = %s)", o.cutoff, time.Unix(cutoffTS, 0).Format(time.RFC3339))
	activity, covers := buildActivityWithStore(client, cutoffTS, o)

	// We can only confirm a member is inactive if we have activity coverage over
	// the entire cutoff window. If we don't, absence from the map is ambiguous and
	// kicking would be unsafe, so refuse to act.
	if !covers {
		abortNoCoverage(activity, o)
	}
	log.Printf("activity map: %d distinct users, oldest activity %s", len(activity.lastActive), time.Unix(activity.oldestSeen, 0).Format(time.RFC3339))

	users := usersByID(client)
	channelsByName := publicChannelsByName(client)
	guarded := toSet(config.GuardedChannels)
	allow := toSet(splitList(o.allow))

	if o.dryRun {
		log.Printf("DRY RUN: no users will be kicked. Pass --dry-run=false to act.")
	}

	kicks := 0
	capped := false
	for _, name := range channelNames {
		conv, ok := channelsByName[name]
		if !ok {
			log.Printf("channel %q: not found among public channels, skipping", name)
			continue
		}
		if guarded[name] || guarded[conv.ID] {
			log.Printf("channel %q: guarded, skipping", name)
			continue
		}
		if conv.IsGeneral {
			log.Printf("channel %q: is #general (cannot kick), skipping", name)
			continue
		}

		members, err := client.GetConversationMembers(conv.ID)
		if err != nil {
			log.Printf("channel %q: failed to list members: %v", name, err)
			continue
		}

		candidates := inactiveCandidates(members, users, activity, cutoffTS, allow)
		log.Printf("channel %q (%s): %d members, %d inactive candidates", name, conv.ID, len(members), len(candidates))

		for _, id := range candidates {
			if kicks >= o.maxKicks {
				capped = true
				break
			}
			last, seen := activity.lastActive[id]
			lastStr := "never"
			if seen {
				lastStr = time.Unix(last, 0).Format("2006-01-02")
			}
			username := users[id].Name

			if o.dryRun {
				log.Printf("  [dry-run] would kick %s (%s) from %q; last active %s", id, username, name, lastStr)
				kicks++
				continue
			}

			if err := client.KickFromConversation(conv.ID, id); err != nil {
				if e, ok := err.(slack.ErrSlack); ok && (e.Type == "not_in_channel" || e.Type == "cant_kick_from_general" || e.Type == "cant_kick_self") {
					log.Printf("  skipped %s (%s) from %q: %s", id, username, name, e.Type)
					continue
				}
				log.Printf("  ERROR kicking %s (%s) from %q: %v", id, username, name, err)
				continue
			}
			log.Printf("  kicked %s (%s) from %q; last active %s", id, username, name, lastStr)
			kicks++
		}
		if capped {
			break
		}
	}

	verb := "would kick"
	if !o.dryRun {
		verb = "kicked"
	}
	log.Printf("=== channel-kick done: %s %d users ===", verb, kicks)
	if capped {
		log.Printf("NOTE: hit the --max-kicks cap (%d); more candidates remain. Re-run to continue.", o.maxKicks)
	}
}

// inactiveCandidates returns the member IDs that are safe to prune.
func inactiveCandidates(members []string, users map[string]slack.User, activity walkStats, cutoffTS int64, allow map[string]bool) []string {
	var candidates []string
	for _, id := range members {
		u, known := users[id]
		// If we can't see the user's flags, protect them rather than guess.
		if !known {
			continue
		}
		if pruneCandidate(u, activity, cutoffTS, allow) {
			candidates = append(candidates, id)
		}
	}
	return candidates
}

// pruneCandidate reports whether a known account should be pruned: not a
// protected account, not on the allowlist, and with no activity at or after the
// cutoff. This is the single decision point shared by channel-kick and
// deactivate.
func pruneCandidate(u slack.User, activity walkStats, cutoffTS int64, allow map[string]bool) bool {
	if isProtected(u) {
		return false
	}
	if allow[u.ID] || allow[u.Name] {
		return false
	}
	if last, seen := activity.lastActive[u.ID]; seen && last >= cutoffTS {
		return false // active within the window
	}
	return true
}

// isProtected reports whether an account should never be pruned automatically.
func isProtected(u slack.User) bool {
	return u.IsAdmin ||
		u.IsOwner ||
		u.IsPrimaryOwner ||
		u.IsBot ||
		u.IsAppUser ||
		u.IsRestricted ||
		u.IsUltraRestricted ||
		u.Deleted
}

func usersByID(client *slack.Client) map[string]slack.User {
	users, err := client.GetUsers()
	if err != nil {
		log.Fatalf("failed to list users: %v", err)
	}
	m := make(map[string]slack.User, len(users))
	for _, u := range users {
		m[u.ID] = u
	}
	return m
}

func publicChannelsByName(client *slack.Client) map[string]slack.Conversation {
	channels, err := client.GetPublicChannels()
	if err != nil {
		log.Fatalf("failed to list public channels: %v", err)
	}
	m := make(map[string]slack.Conversation, len(channels))
	for _, c := range channels {
		m[c.Name] = c
	}
	return m
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, i := range items {
		m[i] = true
	}
	return m
}
