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
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

// storeVersion is bumped if the on-disk format changes incompatibly.
const storeVersion = 1

// activityStore is the durable, cutoff-independent record of when each account
// was last active. It is what makes recurring runs cheap: instead of re-walking
// years of access logs every time, a run fetches only the logs newer than
// HighWater and merges them in.
//
//   - Users maps a Slack user ID to their most recent activity (Unix seconds).
//   - HighWater is the point up to which we have fetched; the next run only needs
//     logs newer than this.
//   - CoversSince is the oldest point from which we have complete, continuous
//     coverage (established by the initial backfill). Inactivity over a window is
//     only trustworthy back to here.
type activityStore struct {
	Version     int              `json:"version"`
	CoversSince int64            `json:"coversSince"`
	HighWater   int64            `json:"highWater"`
	Users       map[string]int64 `json:"users"`
}

// covers reports whether the store can be trusted to confirm inactivity over the
// window [cutoffTS, now]: it must have been backfilled at least as far back as
// the cutoff.
func (s *activityStore) covers(cutoffTS int64) bool {
	return s != nil && s.CoversSince != 0 && s.CoversSince <= cutoffTS
}

// loadStore reads an activity store from a local path or a gs:// URL. It returns
// an empty (non-nil) store if the location does not exist yet, so the first run
// starts from scratch.
func loadStore(location string) (*activityStore, error) {
	data, found, err := readObject(location)
	if err != nil {
		return nil, err
	}
	s := &activityStore{Version: storeVersion, Users: map[string]int64{}}
	if !found {
		return s, nil
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Users == nil {
		s.Users = map[string]int64{}
	}
	return s, nil
}

// saveStore writes an activity store to a local path or a gs:// URL.
func saveStore(location string, s *activityStore) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return writeObject(location, data)
}

// readObject fetches bytes from a local path or gs:// URL. found is false (with a
// nil error) when the object does not exist.
func readObject(location string) (data []byte, found bool, err error) {
	if bucket, object, ok := parseGSURL(location); ok {
		return gcsGet(bucket, object)
	}
	b, err := os.ReadFile(location)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// writeObject stores bytes at a local path (atomically) or a gs:// URL.
func writeObject(location string, data []byte) error {
	if bucket, object, ok := parseGSURL(location); ok {
		return gcsPut(bucket, object, data)
	}
	tmp := location + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, location)
}

// parseGSURL splits a gs://bucket/object URL. ok is false for non-gs:// strings.
func parseGSURL(location string) (bucket, object string, ok bool) {
	const prefix = "gs://"
	if !strings.HasPrefix(location, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(location, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// abortNoCoverage ends the run when we can't confirm inactivity over the cutoff
// window, explaining which reason applies.
func abortNoCoverage(activity walkStats, o options) {
	switch {
	case activity.paused:
		log.Fatalf("aborting: access-log walk paused at the per-run cap (%d) before completing; re-run until the backfill finishes, then act", o.maxAPICalls)
	case activity.hitCallCap:
		log.Fatalf("aborting: access-log walk hit the --max-api-calls cap (%d) before completing, so inactivity can't be confirmed; raise the cap or set --checkpoint and retry", o.maxAPICalls)
	default:
		log.Fatalf("aborting: activity coverage does not reach the %s cutoff (oldest entry %s), so inactivity can't be confirmed for that long a window", o.cutoff, fmtTime(activity.oldestSeen))
	}
}

// buildActivityWithStore produces the last-active map that the acting modes use,
// backed by the durable store at o.storeLocation. With no store location it
// falls back to a full walk (gated on reaching the cutoff). With one it loads the
// store, fetches only the logs newer than the stored high-water mark (a full
// backfill the first time), merges, persists, and reports whether the store now
// covers the whole cutoff window.
func buildActivityWithStore(client *slack.Client, cutoffTS int64, o options) (walkStats, bool) {
	if o.storeLocation == "" {
		stats := buildActivity(client, cutoffTS, o, nil)
		return stats, stats.reachedCutoff
	}

	store, err := loadStore(o.storeLocation)
	if err != nil {
		log.Fatalf("failed to load activity store %s: %v", o.storeLocation, err)
	}

	// Incremental when we already have coverage: only fetch newer than the
	// high-water mark. Otherwise fetch all the way back to the cutoff (backfill).
	floorTS := cutoffTS
	incremental := store.HighWater > 0
	if incremental {
		floorTS = store.HighWater
		log.Printf("activity store: %d users, covers since %s, high-water %s; fetching logs since high-water", len(store.Users), fmtDate(store.CoversSince), fmtDate(store.HighWater))
	} else {
		log.Printf("activity store empty; backfilling access logs to the cutoff %s", fmtDate(cutoffTS))
	}

	runStart := time.Now().Unix()
	stats := buildActivity(client, floorTS, o, store.Users)

	// Persist whatever we learned, even if the walk didn't complete (the merged
	// timestamps are still valid maxes; re-running just re-fetches the tail).
	store.Users = stats.lastActive
	walkCompleted := stats.reachedCutoff || stats.exhausted
	if walkCompleted {
		store.HighWater = runStart
		if store.CoversSince == 0 {
			// First backfill: coverage extends back as far as we reached.
			store.CoversSince = stats.oldestSeen
		}
	}
	if err := saveStore(o.storeLocation, store); err != nil {
		log.Fatalf("failed to save activity store %s: %v", o.storeLocation, err)
	}

	return stats, walkCompleted && store.covers(cutoffTS)
}
