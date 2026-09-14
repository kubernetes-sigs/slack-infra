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
)

func TestParseGSURL(t *testing.T) {
	cases := []struct {
		in     string
		bucket string
		object string
		ok     bool
	}{
		{"gs://my-bucket/slack-prune/activity.json", "my-bucket", "slack-prune/activity.json", true},
		{"gs://bucket/obj", "bucket", "obj", true},
		{"/local/path/activity.json", "", "", false},
		{"gs://bucket-only", "", "", false},
		{"gs://", "", "", false},
		{"gs:///object", "", "", false},
	}
	for _, c := range cases {
		b, o, ok := parseGSURL(c.in)
		if ok != c.ok || b != c.bucket || o != c.object {
			t.Errorf("parseGSURL(%q) = (%q,%q,%v), want (%q,%q,%v)", c.in, b, o, ok, c.bucket, c.object, c.ok)
		}
	}
}

func TestStoreCovers(t *testing.T) {
	cutoff := int64(1000)
	cases := []struct {
		name  string
		store *activityStore
		want  bool
	}{
		{"nil store", nil, false},
		{"never backfilled", &activityStore{CoversSince: 0}, false},
		{"covers before cutoff", &activityStore{CoversSince: 999}, true},
		{"covers exactly at cutoff", &activityStore{CoversSince: 1000}, true},
		{"does not reach cutoff", &activityStore{CoversSince: 1001}, false},
	}
	for _, c := range cases {
		if got := c.store.covers(cutoff); got != c.want {
			t.Errorf("%s: covers(%d) = %v, want %v", c.name, cutoff, got, c.want)
		}
	}
}

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.json")

	// Missing store loads as empty, non-nil, with a usable map.
	s, err := loadStore(path)
	if err != nil {
		t.Fatalf("loadStore (missing): %v", err)
	}
	if s == nil || s.Users == nil {
		t.Fatalf("loadStore (missing) should return an empty non-nil store with a map, got %+v", s)
	}

	s.CoversSince = 100
	s.HighWater = 500
	s.Users["U1"] = 400
	if err := saveStore(path, s); err != nil {
		t.Fatalf("saveStore: %v", err)
	}

	got, err := loadStore(path)
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	if !reflect.DeepEqual(got, s) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, s)
	}
}
