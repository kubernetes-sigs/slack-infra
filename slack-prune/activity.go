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

// walkStats captures what the access-log walk observed: the per-user last-active
// map that drives pruning, plus enough metadata to judge whether the walk can be
// trusted to have covered the whole cutoff window.
type walkStats struct {
	apiCalls      int
	totalEntries  int
	lastActive    map[string]int64  // user ID -> most recent activity (Unix seconds)
	usernames     map[string]string // user ID -> username, for reporting
	oldestSeen    int64             // oldest date_last observed across all entries
	reachedCutoff bool              // true if we paged all the way past the cutoff
	hitCallCap    bool              // true if we bailed on the call cap (no checkpoint)
	exhausted     bool              // true if Slack ran out of logs before the cutoff
	paused        bool              // true if we hit the per-run call cap but checkpointed
}

// buildActivity walks team.accessLogs backwards until it passes the cutoff, runs
// out of logs, or trips the per-invocation call cap, returning the observed
// last-active map and metadata.
//
// The walk is a single loop over a cursor of (before, page, windowMin):
//   - before   is the current query anchor; 0 means "most recent". We page
//     through it, then re-anchor to the oldest entry seen to go further back
//     (working around the 100-page-per-query limit).
//   - page      is the next page to fetch within the current anchor.
//   - windowMin is the oldest date_last seen so far within the current anchor.
//
// When --checkpoint is set the cursor and accumulated results are persisted
// periodically, so a walk that is killed (crash, eviction, rate-limit timeout)
// resumes from where it left off rather than restarting. That is what makes a
// full, multi-hour walk actually completable.
// It walks back only as far as floorTS: pass the cutoff for a full backfill, or
// a stored high-water mark for an incremental fetch. seed pre-populates the
// last-active map (e.g. from an existing activity store) so an incremental walk
// merges into prior results.
func buildActivity(client *slack.Client, floorTS int64, o options, seed map[string]int64) walkStats {
	stats := walkStats{
		lastActive: map[string]int64{},
		usernames:  map[string]string{},
	}
	for id, ts := range seed {
		stats.lastActive[id] = ts
	}
	var before, windowMin int64
	page := 1

	// Resume from a checkpoint if one is configured and present.
	if o.checkpointPath != "" {
		cp, err := loadCheckpoint(o.checkpointPath)
		if err != nil {
			log.Fatalf("failed to read checkpoint %s: %v", o.checkpointPath, err)
		}
		if cp != nil {
			if cp.CutoffTS != floorTS || cp.PageSize != o.pageSize {
				log.Fatalf("checkpoint %s was built with a different --cutoff or --page-size; delete it to start over", o.checkpointPath)
			}
			stats.apiCalls = cp.APICalls
			stats.totalEntries = cp.TotalEntries
			stats.oldestSeen = cp.OldestSeen
			stats.reachedCutoff = cp.ReachedCutoff
			stats.exhausted = cp.Exhausted
			if cp.LastActive != nil {
				stats.lastActive = cp.LastActive
			}
			if cp.Usernames != nil {
				stats.usernames = cp.Usernames
			}
			before, page, windowMin = cp.Before, cp.Page, cp.WindowMin
			if cp.Complete {
				log.Printf("checkpoint is complete; reporting cached results (%d users, %d API calls)", len(stats.lastActive), stats.apiCalls)
				return stats
			}
			log.Printf("resuming from checkpoint: %d API calls, %d users, oldest seen %s", stats.apiCalls, len(stats.lastActive), fmtDate(stats.oldestSeen))
		}
	}

	save := func(complete bool) {
		if o.checkpointPath == "" {
			return
		}
		cp := &walkCheckpoint{
			Version:       checkpointVersion,
			CutoffTS:      floorTS,
			PageSize:      o.pageSize,
			Before:        before,
			Page:          page,
			WindowMin:     windowMin,
			APICalls:      stats.apiCalls,
			TotalEntries:  stats.totalEntries,
			OldestSeen:    stats.oldestSeen,
			ReachedCutoff: stats.reachedCutoff,
			Exhausted:     stats.exhausted,
			Complete:      complete,
			LastActive:    stats.lastActive,
			Usernames:     stats.usernames,
		}
		if err := saveCheckpoint(o.checkpointPath, cp); err != nil {
			log.Fatalf("failed to write checkpoint %s: %v", o.checkpointPath, err)
		}
	}

	callsThisRun := 0
	for {
		// Persist periodically. The cursor here points at the (before, page) we
		// are about to fetch, so this is always a valid resume point.
		if o.checkpointEvery > 0 && callsThisRun > 0 && callsThisRun%o.checkpointEvery == 0 {
			save(false)
		}

		res, err := client.GetAccessLogsPage(before, o.pageSize, page)
		if err != nil {
			log.Fatalf("access log walk failed: %v", err)
		}
		stats.apiCalls++
		callsThisRun++

		if len(res.Logins) == 0 {
			if page == 1 {
				// Nothing older than this anchor exists: history is exhausted
				// (unless we had already reached the floor, handled below).
				if stats.oldestSeen != 0 && stats.oldestSeen <= floorTS {
					stats.reachedCutoff = true
				} else {
					stats.exhausted = true
				}
				break
			}
			// This anchor had fewer pages than expected; re-anchor to continue.
			if !advance(&before, &page, &windowMin, &stats) {
				break
			}
			continue
		}

		for _, l := range res.Logins {
			stats.totalEntries++
			if l.DateLast > stats.lastActive[l.UserID] {
				stats.lastActive[l.UserID] = l.DateLast
				stats.usernames[l.UserID] = l.Username
			}
			if stats.oldestSeen == 0 || l.DateLast < stats.oldestSeen {
				stats.oldestSeen = l.DateLast
			}
			if windowMin == 0 || l.DateLast < windowMin {
				windowMin = l.DateLast
			}
		}

		if o.progressN > 0 && stats.apiCalls%o.progressN == 0 {
			log.Printf("  progress: %d API calls, %d entries, %d distinct users, oldest seen %s",
				stats.apiCalls, stats.totalEntries, len(stats.lastActive), fmtDate(stats.oldestSeen))
		}

		// Reached the floor: because entries come newest-first, everything more
		// recent than it has now been seen.
		if stats.oldestSeen != 0 && stats.oldestSeen <= floorTS {
			stats.reachedCutoff = true
			break
		}

		// Per-invocation safety cap. With a checkpoint this is a resumable pause
		// (re-run to continue); without one it is a terminal bail, as before.
		if o.maxAPICalls > 0 && callsThisRun >= o.maxAPICalls {
			if o.checkpointPath != "" {
				stats.paused = true
			} else {
				stats.hitCallCap = true
			}
			break
		}

		if page >= res.Pages || page >= maxPages {
			// Last page available in this anchor; re-anchor to go further back.
			if !advance(&before, &page, &windowMin, &stats) {
				break
			}
			continue
		}
		page++
	}

	save(stats.reachedCutoff || stats.exhausted)
	return stats
}

// advance re-anchors the cursor to the window just older than the current one.
// It returns false (and marks the walk exhausted) when there is nowhere older
// left to go.
func advance(before *int64, page *int, windowMin *int64, stats *walkStats) bool {
	if *windowMin == 0 {
		stats.exhausted = true
		return false
	}
	// `before` is inclusive, so guard against stalling on a cluster of
	// same-second logins by stepping back one second.
	next := *windowMin
	if *before != 0 && next >= *before {
		next = *before - 1
	}
	if next <= 0 {
		stats.exhausted = true
		return false
	}
	*before = next
	*page = 1
	*windowMin = 0
	return true
}

func fmtDate(ts int64) string {
	if ts == 0 {
		return "n/a"
	}
	return time.Unix(ts, 0).Format("2006-01-02")
}

func fmtTime(ts int64) string {
	if ts == 0 {
		return "n/a"
	}
	return time.Unix(ts, 0).Format(time.RFC3339)
}

// runReport walks the access logs and prints volume/runtime only.
func runReport(client *slack.Client, o options) {
	cutoffTS := time.Now().Add(-o.cutoff).Unix()
	log.Printf("Walking team.accessLogs back to %s (cutoff %s ago)", fmtTime(cutoffTS), o.cutoff)

	start := time.Now()
	stats := buildActivity(client, cutoffTS, o, nil)
	elapsed := time.Since(start)

	activeSinceCutoff := 0
	for _, last := range stats.lastActive {
		if last >= cutoffTS {
			activeSinceCutoff++
		}
	}

	log.Printf("=== slack-prune report ===")
	log.Printf("elapsed:                 %s", elapsed.Round(time.Second))
	log.Printf("API calls:               %d", stats.apiCalls)
	log.Printf("access-log entries read: %d", stats.totalEntries)
	log.Printf("distinct users seen:     %d", len(stats.lastActive))
	log.Printf("active since cutoff:     %d", activeSinceCutoff)
	log.Printf("oldest activity reached: %s", fmtTime(stats.oldestSeen))

	switch {
	case stats.reachedCutoff:
		log.Printf("outcome: REACHED CUTOFF - the full window is walkable in the numbers above")
	case stats.exhausted:
		log.Printf("outcome: LOGS EXHAUSTED - Slack returned no more entries before reaching the cutoff")
	case stats.paused:
		log.Printf("outcome: PAUSED - hit the per-run cap (%d calls); checkpoint saved, re-run with the same --checkpoint to continue", o.maxAPICalls)
	case stats.hitCallCap:
		log.Printf("outcome: HIT API-CALL CAP (%d) before the cutoff - raise --max-api-calls (or set --checkpoint to make it resumable)", o.maxAPICalls)
	}

	if o.dumpActive {
		log.Printf("--- users active since cutoff ---")
		for id, last := range stats.lastActive {
			if last >= cutoffTS {
				log.Printf("%s (%s): %s", id, stats.usernames[id], fmtTime(last))
			}
		}
	}
}
