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
	"reflect"
	"testing"

	"sigs.k8s.io/slack-infra/slack"
)

const cutoffTS = int64(1_000_000)

// activityWith builds a walkStats whose last-active map holds the given entries.
func activityWith(lastActive map[string]int64) walkStats {
	return walkStats{lastActive: lastActive, usernames: map[string]string{}}
}

func TestIsProtected(t *testing.T) {
	cases := []struct {
		name string
		user slack.User
		want bool
	}{
		{"plain user", slack.User{ID: "U1"}, false},
		{"admin", slack.User{ID: "U1", IsAdmin: true}, true},
		{"owner", slack.User{ID: "U1", IsOwner: true}, true},
		{"primary owner", slack.User{ID: "U1", IsPrimaryOwner: true}, true},
		{"bot", slack.User{ID: "U1", IsBot: true}, true},
		{"app user", slack.User{ID: "U1", IsAppUser: true}, true},
		{"restricted (guest)", slack.User{ID: "U1", IsRestricted: true}, true},
		{"ultra restricted", slack.User{ID: "U1", IsUltraRestricted: true}, true},
		{"already deactivated", slack.User{ID: "U1", Deleted: true}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isProtected(c.user); got != c.want {
				t.Errorf("isProtected(%s) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestPruneCandidate(t *testing.T) {
	activity := activityWith(map[string]int64{
		"active":   cutoffTS + 1, // after cutoff -> active
		"onCutoff": cutoffTS,     // exactly at cutoff counts as active
		"stale":    cutoffTS - 1, // before cutoff -> inactive
	})
	// "unseen" never appears in the activity map at all.

	cases := []struct {
		name  string
		user  slack.User
		allow map[string]bool
		want  bool
	}{
		{"active user is kept", slack.User{ID: "active"}, nil, false},
		{"activity exactly at cutoff is kept", slack.User{ID: "onCutoff"}, nil, false},
		{"stale user is a candidate", slack.User{ID: "stale"}, nil, true},
		{"unseen user is a candidate", slack.User{ID: "unseen"}, nil, true},
		{"protected admin is kept even if stale", slack.User{ID: "stale", IsAdmin: true}, nil, false},
		{"allowlisted by ID is kept", slack.User{ID: "stale"}, map[string]bool{"stale": true}, false},
		{"allowlisted by name is kept", slack.User{ID: "stale", Name: "keepme"}, map[string]bool{"keepme": true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pruneCandidate(c.user, activity, cutoffTS, c.allow); got != c.want {
				t.Errorf("pruneCandidate(%s) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestInactiveCandidates(t *testing.T) {
	activity := activityWith(map[string]int64{
		"active": cutoffTS + 1,
		"stale":  cutoffTS - 1,
	})
	users := map[string]slack.User{
		"active": {ID: "active"},
		"stale":  {ID: "stale"},
		"admin":  {ID: "admin", IsAdmin: true},
		"unseen": {ID: "unseen"},
	}
	// "ghost" is a channel member with no entry in the users map at all.
	members := []string{"active", "stale", "admin", "unseen", "ghost"}

	got := inactiveCandidates(members, users, activity, cutoffTS, nil)
	want := []string{"stale", "unseen"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("inactiveCandidates() = %v, want %v (ghost must be protected as unknown; admin/active must be excluded)", got, want)
	}
}

func TestSplitList(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"  ", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b ,, c ", []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		if got := splitList(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitList(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestToSet(t *testing.T) {
	got := toSet([]string{"a", "b", "a"})
	if len(got) != 2 || !got["a"] || !got["b"] {
		t.Errorf("toSet() = %v, want set{a,b}", got)
	}
	if toSet(nil)["anything"] {
		t.Errorf("empty set should not contain arbitrary keys")
	}
}
