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
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestMembersCacheRoundTripAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "members.json")

	// An empty location disables caching; every method must be nil-safe.
	off, err := loadMembersCache("")
	if err != nil {
		t.Fatalf("loadMembersCache(\"\"): %v", err)
	}
	if off != nil {
		t.Fatalf("empty location should disable the cache, got %+v", off)
	}
	off.remove("C1", map[string]bool{"U1": true})
	off.save()

	c, err := loadMembersCache(path)
	if err != nil {
		t.Fatalf("loadMembersCache (missing): %v", err)
	}
	if c == nil || c.Channels == nil {
		t.Fatalf("missing cache should load empty and usable, got %+v", c)
	}

	c.Channels["C1"] = cachedMembers{Fetched: time.Now().Unix(), Members: []string{"U1", "U2", "U3"}}
	c.remove("C1", map[string]bool{"U2": true})
	if got, want := c.Channels["C1"].Members, []string{"U1", "U3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after remove: got %v, want %v", got, want)
	}
	c.save()

	got, err := loadMembersCache(path)
	if err != nil {
		t.Fatalf("loadMembersCache: %v", err)
	}
	if !reflect.DeepEqual(got.Channels, c.Channels) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got.Channels, c.Channels)
	}
}

func TestMembersCacheFreshness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "members.json")
	c, err := loadMembersCache(path)
	if err != nil {
		t.Fatalf("loadMembersCache: %v", err)
	}

	now := time.Now().Unix()
	c.Channels["C1"] = cachedMembers{Fetched: now - 3600, Members: []string{"U1"}}

	// Fresh entry: served from the cache without touching the (nil) client.
	members, err := c.members(nil, "C1", 24*time.Hour)
	if err != nil {
		t.Fatalf("members (fresh): %v", err)
	}
	if !reflect.DeepEqual(members, []string{"U1"}) {
		t.Errorf("members (fresh) = %v, want [U1]", members)
	}

	// Stale entry: must go back to the API. A nil client panics if it does,
	// which is what proves the cache was bypassed.
	defer func() {
		if recover() == nil {
			t.Error("stale entry was served from the cache; expected a fresh listing")
		}
	}()
	_, _ = c.members(nil, "C1", 30*time.Minute)
}
