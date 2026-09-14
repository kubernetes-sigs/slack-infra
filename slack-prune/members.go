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
	"log"
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

// membersCacheVersion is bumped if the on-disk format changes incompatibly.
const membersCacheVersion = 1

// membersCache is a local copy of channel membership, keyed by channel ID.
// Listing the members of a large channel (#kubernetes-users runs to tens of
// thousands) takes minutes of cursor paging, which dominates the runtime of a
// repeated channel-kick run. Kicks are applied to the cached list as they
// happen, so a re-run picks up where the last one left off without re-listing.
//
// Membership drifts between runs (people join and leave), so an entry is only
// used while it is younger than the TTL. A member who left on their own is
// still in the cache until it expires; kicking them returns not_in_channel,
// which the caller already skips.
type membersCache struct {
	Version  int                      `json:"version"`
	Channels map[string]cachedMembers `json:"channels"`

	location string
}

type cachedMembers struct {
	Fetched int64    `json:"fetched"`
	Members []string `json:"members"`
}

// loadMembersCache reads the cache from a local path or gs:// URL. An empty
// location returns a nil cache, which disables caching: every method is a
// pass-through on nil.
func loadMembersCache(location string) (*membersCache, error) {
	if location == "" {
		return nil, nil
	}
	data, found, err := readObject(location)
	if err != nil {
		return nil, err
	}
	c := &membersCache{Version: membersCacheVersion, Channels: map[string]cachedMembers{}, location: location}
	if !found {
		return c, nil
	}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	if c.Version != membersCacheVersion {
		// Format we don't understand: start over rather than misread it.
		return &membersCache{Version: membersCacheVersion, Channels: map[string]cachedMembers{}, location: location}, nil
	}
	if c.Channels == nil {
		c.Channels = map[string]cachedMembers{}
	}
	c.location = location
	return c, nil
}

// members returns the member IDs of channelID, from the cache when the entry is
// younger than ttl and from the API otherwise.
func (c *membersCache) members(client *slack.Client, channelID string, ttl time.Duration) ([]string, error) {
	if c == nil {
		return client.GetConversationMembers(channelID)
	}
	now := time.Now().Unix()
	if e, ok := c.Channels[channelID]; ok && now-e.Fetched < int64(ttl.Seconds()) {
		log.Printf("channel %s: using %d cached members from %s", channelID, len(e.Members), fmtTime(e.Fetched))
		return e.Members, nil
	}
	members, err := client.GetConversationMembers(channelID)
	if err != nil {
		return nil, err
	}
	c.Channels[channelID] = cachedMembers{Fetched: now, Members: members}
	return members, nil
}

// remove drops IDs from a channel's cached membership, keeping the cache in
// step with the kicks that were just performed.
func (c *membersCache) remove(channelID string, ids map[string]bool) {
	if c == nil || len(ids) == 0 {
		return
	}
	e, ok := c.Channels[channelID]
	if !ok {
		return
	}
	kept := make([]string, 0, len(e.Members))
	for _, id := range e.Members {
		if !ids[id] {
			kept = append(kept, id)
		}
	}
	e.Members = kept
	c.Channels[channelID] = e
}

// save persists the cache. Failure is logged, not fatal: a stale cache costs a
// re-list on the next run, which is not worth losing a completed set of kicks
// over.
func (c *membersCache) save() {
	if c == nil {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		log.Printf("failed to encode members cache: %v", err)
		return
	}
	if err := writeObject(c.location, data); err != nil {
		log.Printf("failed to save members cache %s: %v", c.location, err)
	}
}
