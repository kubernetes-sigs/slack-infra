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

func TestLoadCheckpointMissing(t *testing.T) {
	cp, err := loadCheckpoint(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("loadCheckpoint on a missing file should not error, got %v", err)
	}
	if cp != nil {
		t.Fatalf("loadCheckpoint on a missing file should return nil, got %+v", cp)
	}
}

func TestCheckpointRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cp.json")
	want := &walkCheckpoint{
		Version:      checkpointVersion,
		CutoffTS:     1000,
		PageSize:     1000,
		Before:       12345,
		Page:         7,
		WindowMin:    9999,
		APICalls:     42,
		TotalEntries: 41000,
		OldestSeen:   500,
		LastActive:   map[string]int64{"U1": 900, "U2": 100},
		Usernames:    map[string]string{"U1": "alice", "U2": "bob"},
	}
	if err := saveCheckpoint(path, want); err != nil {
		t.Fatalf("saveCheckpoint: %v", err)
	}
	got, err := loadCheckpoint(path)
	if err != nil {
		t.Fatalf("loadCheckpoint: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestAdvance(t *testing.T) {
	cases := []struct {
		name          string
		before        int64
		windowMin     int64
		wantOK        bool
		wantBefore    int64
		wantExhausted bool
	}{
		{"first re-anchor from newest", 0, 500, true, 500, false},
		{"re-anchor further back", 800, 400, true, 400, false},
		{"step back one on non-decreasing windowMin", 500, 500, true, 499, false},
		{"exhausted when no window seen", 800, 0, false, 800, true},
		{"exhausted when stepping below zero", 1, 1, false, 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before, windowMin := c.before, c.windowMin
			page := 9
			stats := &walkStats{}
			ok := advance(&before, &page, &windowMin, stats)
			if ok != c.wantOK {
				t.Fatalf("advance ok = %v, want %v", ok, c.wantOK)
			}
			if stats.exhausted != c.wantExhausted {
				t.Errorf("exhausted = %v, want %v", stats.exhausted, c.wantExhausted)
			}
			if !c.wantOK {
				return
			}
			if before != c.wantBefore {
				t.Errorf("before = %d, want %d", before, c.wantBefore)
			}
			if page != 1 {
				t.Errorf("page reset = %d, want 1", page)
			}
			if windowMin != 0 {
				t.Errorf("windowMin reset = %d, want 0", windowMin)
			}
		})
	}
}
