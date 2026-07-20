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

// runDeactivate deactivates long-inactive accounts workspace-wide via the admin
// users.admin.setInactive endpoint. It defaults to a dry run; pass
// --dry-run=false to actually deactivate.
func runDeactivate(client *slack.Client, o options) {
	cutoffTS := time.Now().Add(-o.cutoff).Unix()
	log.Printf("Building activity map (cutoff %s ago = %s)", o.cutoff, time.Unix(cutoffTS, 0).Format(time.RFC3339))
	activity, covers := buildActivityWithStore(client, cutoffTS, o)

	// Same safety gate as channel-kick: only act if we have activity coverage over
	// the whole cutoff window, so absence from the map genuinely means "inactive".
	if !covers {
		abortNoCoverage(activity, o)
	}
	log.Printf("activity map: %d distinct users, oldest activity %s", len(activity.lastActive), time.Unix(activity.oldestSeen, 0).Format(time.RFC3339))

	users, err := client.GetUsers()
	if err != nil {
		log.Fatalf("failed to list users: %v", err)
	}
	allow := toSet(splitList(o.allow))

	if o.dryRun {
		log.Printf("DRY RUN: no users will be deactivated. Pass --dry-run=false to act.")
	}

	candidates := 0
	done := 0
	capped := false
	for _, u := range users {
		if !pruneCandidate(u, activity, cutoffTS, allow) {
			continue
		}
		candidates++

		if done >= o.maxDeactivations {
			capped = true
			continue
		}

		lastStr := "never"
		if last, seen := activity.lastActive[u.ID]; seen {
			lastStr = time.Unix(last, 0).Format("2006-01-02")
		}

		if o.dryRun {
			log.Printf("  [dry-run] would deactivate %s (%s); last active %s", u.ID, u.Name, lastStr)
			done++
			continue
		}

		if err := client.SetUserInactive(u.ID); err != nil {
			log.Printf("  ERROR deactivating %s (%s): %v", u.ID, u.Name, err)
			continue
		}
		log.Printf("  deactivated %s (%s); last active %s", u.ID, u.Name, lastStr)
		done++
	}

	verb := "would deactivate"
	if !o.dryRun {
		verb = "deactivated"
	}
	log.Printf("=== deactivate done: %d inactive candidates, %s %d ===", candidates, verb, done)
	if capped {
		log.Printf("NOTE: hit the --max-deactivations cap (%d); %d candidates remain. Re-run to continue.", o.maxDeactivations, candidates-done)
	}
}
