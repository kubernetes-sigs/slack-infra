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

package reconciler

import (
	"reflect"
	"testing"

	"sigs.k8s.io/slack-infra/slack"
	"sigs.k8s.io/slack-infra/tempelis/config"
)

func TestReconcileUsergroups(t *testing.T) {
	userMapping := map[string]string{
		"Katharine":   "U12345678",
		"bentheelder": "U11111111",
	}

	tests := []struct {
		name             string
		priorGroups      []slack.Subteam
		priorChannels    []slack.Conversation
		newGroups        []config.Usergroup
		expectedActions  []Action
		expectedErrCount int
	}{
		{
			name:      "creating a new simple group",
			newGroups: []config.Usergroup{{Name: "pony-fans", LongName: "Pony Fans", Description: "Fans of ponies", Members: []string{"Katharine"}}},
			expectedActions: []Action{
				updateUsergroupAction{handle: "pony-fans", name: "Pony Fans", description: "Fans of ponies", create: true},
				updateUsergroupMembersAction{name: "pony-fans", users: []string{"U12345678"}},
			},
		},
		{
			name:            "removing a group",
			priorGroups:     []slack.Subteam{{Handle: "pony-fans", ID: "S12345678"}},
			newGroups:       nil,
			expectedActions: []Action{deactivateUsergroupAction{id: "S12345678", handle: "pony-fans"}},
		},
		{
			name:            "updating a group's long name",
			priorGroups:     []slack.Subteam{{Handle: "pony-fans", ID: "S12345678", Name: "Pon", Description: "Fans of ponies", Users: []string{"U12345678"}}},
			newGroups:       []config.Usergroup{{Name: "pony-fans", LongName: "Pony Fans", Description: "Fans of ponies", Members: []string{"Katharine"}}},
			expectedActions: []Action{updateUsergroupAction{id: "S12345678", handle: "pony-fans", name: "Pony Fans", description: "Fans of ponies"}},
		},
		{
			name:            "updating a group's description",
			priorGroups:     []slack.Subteam{{Handle: "pony-fans", ID: "S12345678", Name: "Pony Fans", Description: "an old description", Users: []string{"U12345678"}}},
			newGroups:       []config.Usergroup{{Name: "pony-fans", LongName: "Pony Fans", Description: "Fans of ponies", Members: []string{"Katharine"}}},
			expectedActions: []Action{updateUsergroupAction{id: "S12345678", handle: "pony-fans", name: "Pony Fans", description: "Fans of ponies"}},
		},
		{
			name:            "updating a group's channel list",
			priorChannels:   []slack.Conversation{{Name: "pony-channel"}},
			priorGroups:     []slack.Subteam{{Handle: "pony-fans", ID: "S12345678", Name: "Pony Fans", Description: "an old description", Users: []string{"U12345678"}}},
			newGroups:       []config.Usergroup{{Name: "pony-fans", LongName: "Pony Fans", Description: "Fans of ponies", Members: []string{"Katharine"}, Channels: []string{"pony-channel"}}},
			expectedActions: []Action{updateUsergroupAction{id: "S12345678", handle: "pony-fans", name: "Pony Fans", description: "Fans of ponies", channelNames: []string{"pony-channel"}}},
		},
		{
			name:          "doing nothing",
			priorChannels: []slack.Conversation{{Name: "pony-channel", ID: "C22222222"}, {Name: "a-channel", ID: "C11111111"}},
			priorGroups:   []slack.Subteam{{Handle: "pony-fans", ID: "S12345678", Name: "Pony Fans", Description: "Fans of ponies", Users: []string{"U12345678", "U11111111"}, Prefs: slack.SubteamPrefs{Channels: []string{"C11111111", "C22222222"}}}},
			newGroups:     []config.Usergroup{{Name: "pony-fans", LongName: "Pony Fans", Description: "Fans of ponies", Members: []string{"bentheelder", "Katharine"}, Channels: []string{"pony-channel", "a-channel"}}},
		},
		{
			name:            "updating a group's member list",
			priorGroups:     []slack.Subteam{{Handle: "pony-fans", ID: "S12345678", Name: "Pony Fans", Description: "Fans of ponies", Users: []string{"U12345678"}}},
			newGroups:       []config.Usergroup{{Name: "pony-fans", LongName: "Pony Fans", Description: "Fans of ponies", Members: []string{"Katharine", "bentheelder"}}},
			expectedActions: []Action{updateUsergroupMembersAction{id: "S12345678", name: "pony-fans", users: []string{"U11111111", "U12345678"}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Reconciler{
				config:   config.Config{Usergroups: tc.newGroups, Users: userMapping},
				channels: channelState{byID: map[string]*slack.Conversation{}, byName: map[string]*slack.Conversation{}},
				groups:   usergroupState{byID: map[string]*slack.Subteam{}, byHandle: map[string]*slack.Subteam{}},
			}
			for _, c := range tc.priorChannels {
				c2 := c
				r.channels.byID[c.ID] = &c2
				r.channels.byName[c.Name] = &c2
			}
			for _, g := range tc.priorGroups {
				g2 := g
				r.groups.byHandle[g2.Handle] = &g2
				r.groups.byID[g2.ID] = &g2
			}
			actions, errs := r.reconcileUsergroups()
			if !reflect.DeepEqual(actions, tc.expectedActions) {
				t.Errorf("Expected actions: %#v\nActual actions: %#v", tc.expectedActions, actions)
			}
			if len(errs) != tc.expectedErrCount {
				t.Errorf("Expected %d errors, but got %d: %v", tc.expectedErrCount, len(errs), errs)
			}
		})
	}
}
