/*
Copyright 2019 The Kubernetes Authors.
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

package config

import (
	"reflect"
	"regexp"
	"testing"
)

func TestResolveRestrictions(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		restrictions []Restrictions
		expected     Restrictions
	}{
		{
			name:     "no restrictions",
			filePath: "whatever.yaml",
			expected: defaultRestriction,
		},
		{
			name:     "no restrictions match",
			filePath: "not-specified.yaml",
			restrictions: []Restrictions{
				{
					Path: "some-path.yaml",
				},
			},
			expected: defaultRestriction,
		},
		{
			name:     "simple restriction match",
			filePath: "restricted.yaml",
			restrictions: []Restrictions{
				{
					Path:  "restricted.yaml",
					Users: true,
				},
			},
			expected: Restrictions{
				Path:  "restricted.yaml",
				Users: true,
			},
		},
		{
			name:     "falls through to first matching restriction",
			filePath: "second.yaml",
			restrictions: []Restrictions{
				{
					Path:  "first.yaml",
					Users: true,
				},
				{
					Path: "second.yaml",
				},
				{
					Path:     "third.yaml",
					Template: true,
				},
			},
			expected: Restrictions{
				Path: "second.yaml",
			},
		},
		{
			name:     "doublestar matching works",
			filePath: "some/very/nested/path.yaml",
			restrictions: []Restrictions{
				{
					Path:  "**/*.yaml",
					Users: true,
				},
			},
			expected: Restrictions{
				Path:  "**/*.yaml",
				Users: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := resolveRestrictions(tc.restrictions, tc.filePath)
			if !reflect.DeepEqual(r, tc.expected) {
				t.Fatalf("Expected restriction %#v, got %#v", tc.expected, r)
			}
		})
	}
}

func TestMergeRestrictions(t *testing.T) {
	tests := []struct {
		name      string
		a         []Restrictions
		b         []Restrictions
		expected  []Restrictions
		expectErr bool
	}{
		{
			name: "replacing empty restrictions is permitted",
			a:    nil,
			b: []Restrictions{
				{
					Path: "foo.yaml",
				},
				{
					Path: "bar.yaml",
				},
			},
			expected: []Restrictions{
				{
					Path:       "foo.yaml",
					Channels:   []*regexp.Regexp{},
					Usergroups: []*regexp.Regexp{},
				},
				{
					Path:       "bar.yaml",
					Channels:   []*regexp.Regexp{},
					Usergroups: []*regexp.Regexp{},
				},
			},
		},
		{
			name: "replacing existing restrictions is prohibited",
			a: []Restrictions{
				{
					Path: "foo.yaml",
				},
			},
			b: []Restrictions{
				{
					Path:       "bar.yaml",
					Channels:   []*regexp.Regexp{},
					Usergroups: []*regexp.Regexp{},
				},
			},
			expectErr: true,
		},
		{
			name: "not replacing existing restrictions is fine",
			a: []Restrictions{
				{
					Path: "foo.yaml",
				},
				{
					Path: "bar.yaml",
				},
			},
			b: nil,
			expected: []Restrictions{
				{
					Path: "foo.yaml",
				},
				{
					Path: "bar.yaml",
				},
			},
		},
		{
			name: "regexes get parsed",
			a:    nil,
			b: []Restrictions{
				{
					Path:             "foo.yaml",
					ChannelsString:   []string{"foo.*"},
					UsergroupsString: []string{"bar.*"},
				},
			},
			expected: []Restrictions{
				{
					Path:             "foo.yaml",
					ChannelsString:   []string{"foo.*"},
					UsergroupsString: []string{"bar.*"},
					Channels:         []*regexp.Regexp{regexp.MustCompile("foo.*")},
					Usergroups:       []*regexp.Regexp{regexp.MustCompile("bar.*")},
				},
			},
		},
		{
			name: "invalid channel regexes are an error",
			a:    nil,
			b: []Restrictions{
				{
					Path:             "foo.yaml",
					ChannelsString:   []string{"foo("},
					UsergroupsString: []string{"bar.*"},
				},
			},
			expectErr: true,
		},
		{
			name: "invalid usergroup regexes are an error",
			a:    nil,
			b: []Restrictions{
				{
					Path:             "foo.yaml",
					ChannelsString:   []string{"foo"},
					UsergroupsString: []string{"bar("},
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := mergeRestrictions(tc.a, tc.b)
			if err != nil {
				if !tc.expectErr {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.expectErr {
				t.Fatalf("expected an error, but got result %#v", r)
			}
			if !reflect.DeepEqual(r, tc.expected) {
				t.Fatalf("Expected restrictions %#v, got %#v", tc.expected, r)
			}
		})
	}
}

func TestMergeUsers(t *testing.T) {
	tests := []struct {
		name         string
		a            map[string]string
		b            map[string]string
		restrictions Restrictions
		expected     map[string]string
		expectErr    bool
	}{
		{
			name:         "merging into an empty map works",
			a:            map[string]string{},
			b:            map[string]string{"Katharine": "U1234567890", "bentheelder": "U2345678901"},
			restrictions: defaultRestriction,
			expected:     map[string]string{"Katharine": "U1234567890", "bentheelder": "U2345678901"},
		},
		{
			name:         "merging disjoint maps works",
			a:            map[string]string{"Katharine": "U1234567890"},
			b:            map[string]string{"bentheelder": "U2345678901"},
			restrictions: defaultRestriction,
			expected:     map[string]string{"Katharine": "U1234567890", "bentheelder": "U2345678901"},
		},
		{
			name:         "merging overlapping maps is an error",
			a:            map[string]string{"Katharine": "U1234567890"},
			b:            map[string]string{"Katharine": "U1234567901", "bentheelder": "U2345678901"},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
		{
			name:         "merging when not permitted is an error",
			a:            map[string]string{},
			b:            map[string]string{"Katharine": "U1234567890"},
			restrictions: Restrictions{Users: false},
			expectErr:    true,
		},
		{
			name:         "a name without a valid ID is an error",
			a:            map[string]string{},
			b:            map[string]string{"Katharine": "wat"},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
		{
			name:         "doing nothing is fine regardless of permissions",
			a:            map[string]string{"Katharine": "U1234567890"},
			b:            map[string]string{},
			restrictions: Restrictions{Users: false},
			expected:     map[string]string{"Katharine": "U1234567890"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := map[string]string{}
			for k, v := range tc.a {
				a[k] = v
			}
			err := mergeUsers(a, tc.b, tc.restrictions)
			if err != nil {
				if !tc.expectErr {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.expectErr {
				t.Fatalf("expected an error, but got result %#v", a)
			}
			if !reflect.DeepEqual(a, tc.expected) {
				t.Fatalf("Expected users %#v, got %#v", tc.expected, a)
			}
		})
	}
}

func TestMergeChannels(t *testing.T) {
	tests := []struct {
		name         string
		a            []Channel
		b            []Channel
		restrictions Restrictions
		expected     []Channel
		expectErr    bool
	}{
		{
			name:         "merging first set works",
			a:            nil,
			b:            []Channel{{Name: "ponies"}, {Name: "kubernetes"}},
			restrictions: defaultRestriction,
			expected:     []Channel{{Name: "ponies"}, {Name: "kubernetes"}},
		},
		{
			name:         "merging disjoint channels works",
			a:            []Channel{{Name: "slack-admins"}},
			b:            []Channel{{Name: "ponies"}, {Name: "kubernetes"}},
			restrictions: defaultRestriction,
			expected:     []Channel{{Name: "slack-admins"}, {Name: "ponies"}, {Name: "kubernetes"}},
		},
		{
			name:         "merging overlapping channels fails",
			a:            []Channel{{Name: "slack-admins"}, {Name: "ponies"}},
			b:            []Channel{{Name: "ponies"}, {Name: "kubernetes"}},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
		{
			name:         "merging fails when all channels are forbidden",
			a:            []Channel{{Name: "slack-admins"}},
			b:            []Channel{{Name: "ponies"}, {Name: "kubernetes"}},
			restrictions: Restrictions{},
			expectErr:    true,
		},
		{
			name:         "merging fails when no regexes match a new channel",
			a:            []Channel{{Name: "slack-admins"}},
			b:            []Channel{{Name: "ponies"}, {Name: "kubernetes"}},
			restrictions: Restrictions{Channels: []*regexp.Regexp{regexp.MustCompile("ponies?")}},
			expectErr:    true,
		},
		{
			name:         "merging succeeds when a regex matches all new channels",
			a:            []Channel{{Name: "slack-admins"}},
			b:            []Channel{{Name: "ponies"}, {Name: "kubernetes"}},
			restrictions: Restrictions{Channels: []*regexp.Regexp{regexp.MustCompile("ponies?"), regexp.MustCompile("kube.*")}},
			expected:     []Channel{{Name: "slack-admins"}, {Name: "ponies"}, {Name: "kubernetes"}},
		},
		{
			name:         "a channel without a name is illegal",
			a:            nil,
			b:            []Channel{{}},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := mergeChannels(tc.a, tc.b, tc.restrictions)
			if err != nil {
				if !tc.expectErr {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.expectErr {
				t.Fatalf("expected an error, but got result %#v", r)
			}
			if !reflect.DeepEqual(r, tc.expected) {
				t.Fatalf("Expected channels %#v, got %#v", tc.expected, r)
			}
		})
	}
}

func TestMergeUsergroups(t *testing.T) {
	group1 := Usergroup{
		Name:        "pony-fans",
		Members:     []string{"U12345678"},
		LongName:    "Pony Fans",
		Description: "Fans of ponies",
	}
	group2 := Usergroup{
		Name:        "sig-testing",
		Members:     []string{"U11111111", "U22222222", "U33333333"},
		LongName:    "SIG Testing",
		Description: "prow, mostly.",
	}
	tests := []struct {
		name         string
		a            []Usergroup
		b            []Usergroup
		restrictions Restrictions
		expected     []Usergroup
		expectErr    bool
	}{
		{
			name:         "merging first set works",
			a:            nil,
			b:            []Usergroup{group1, group2},
			restrictions: defaultRestriction,
			expected:     []Usergroup{group1, group2},
		},
		{
			name:         "merging disjoint sets works",
			a:            []Usergroup{group1},
			b:            []Usergroup{group2},
			restrictions: defaultRestriction,
			expected:     []Usergroup{group1, group2},
		},
		{
			name:      "merging overlapping groups fails",
			a:         []Usergroup{group2},
			b:         []Usergroup{group1, group2},
			expectErr: true,
		},
		{
			name:         "merging fails when all groups are forbidden",
			a:            nil,
			b:            []Usergroup{group1},
			restrictions: Restrictions{},
			expectErr:    true,
		},
		{
			name:         "merging fails when no regexes match a new group",
			a:            nil,
			b:            []Usergroup{group1},
			restrictions: Restrictions{Usergroups: []*regexp.Regexp{regexp.MustCompile("sig.*")}},
			expectErr:    true,
		},
		{
			name:         "merging passes when all groups match a regex",
			a:            nil,
			b:            []Usergroup{group1, group2},
			restrictions: Restrictions{Usergroups: []*regexp.Regexp{regexp.MustCompile("sig.*"), regexp.MustCompile("pony|ponies")}},
			expected:     []Usergroup{group1, group2},
		},
		{
			name: "a usergroup missing a name is an error",
			a:    nil,
			b: []Usergroup{{
				Members:     []string{"U11111111"},
				LongName:    "SIG Testing",
				Description: "prow, mostly.",
			}},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
		{
			name: "a usergroup missing a long name is an error",
			a:    nil,
			b: []Usergroup{{
				Members:     []string{"U11111111"},
				Name:        "sig-testing",
				Description: "prow, mostly.",
			}},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
		{
			name: "a usergroup missing a description is an error",
			a:    nil,
			b: []Usergroup{{
				Members:  []string{"U11111111"},
				Name:     "sig-testing",
				LongName: "SIG Testing",
			}},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
		{
			name: "a usergroup with no members is an error",
			a:    nil,
			b: []Usergroup{{
				Members:     []string{},
				Name:        "sig-testing",
				LongName:    "SIG Testing",
				Description: "prow, mostly.",
			}},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
		{
			name: "an externally-managed usergroup only needs a name",
			a:    nil,
			b: []Usergroup{{
				Name:     "sig-testing",
				External: true,
			}},
			restrictions: defaultRestriction,
			expected:     []Usergroup{{Name: "sig-testing", External: true}},
		},
		{
			name:         "an externally-managed usergroup with no name is an error",
			a:            nil,
			b:            []Usergroup{{External: true}},
			restrictions: defaultRestriction,
			expectErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := mergeUsergroups(tc.a, tc.b, tc.restrictions)
			if err != nil {
				if !tc.expectErr {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.expectErr {
				t.Fatalf("expected an error, but got result %#v", r)
			}
			if !reflect.DeepEqual(r, tc.expected) {
				t.Fatalf("Expected usergroups %#v, got %#v", tc.expected, r)
			}
		})
	}
}
