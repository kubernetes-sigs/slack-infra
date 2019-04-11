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

func TestReconcileChannels(t *testing.T) {
	tests := []struct {
		name             string
		priorChannels    []slack.Conversation
		newChannels      []config.Channel
		expectedActions  []Action
		expectedErrCount int
	}{
		{
			name:            "create a new channel",
			priorChannels:   []slack.Conversation{{Name: "sig-testing", ID: "C12345678"}},
			newChannels:     []config.Channel{{Name: "sig-testing"}, {Name: "sig-contribex"}},
			expectedActions: []Action{createChannelAction{name: "sig-contribex"}},
		},
		{
			name:            "archive a channel",
			priorChannels:   []slack.Conversation{{Name: "sig-testing", ID: "C12345678"}},
			newChannels:     []config.Channel{{Name: "sig-testing", Archived: true}},
			expectedActions: []Action{archiveChannelAction{name: "sig-testing", id: "C12345678"}},
		},
		{
			name:            "unarchive a channel",
			priorChannels:   []slack.Conversation{{Name: "sig-testing", ID: "C12345678", IsArchived: true}},
			newChannels:     []config.Channel{{Name: "sig-testing"}},
			expectedActions: []Action{unarchiveChannelAction{name: "sig-testing", id: "C12345678"}},
		},
		{
			name:          "do nothing to a channel that both is and should be archived",
			priorChannels: []slack.Conversation{{Name: "sig-testing", ID: "C12345678", IsArchived: true}},
			newChannels:   []config.Channel{{Name: "sig-testing", Archived: true}},
		},
		{
			name:             "an extant channel not being mentioned is an error",
			priorChannels:    []slack.Conversation{{Name: "sig-testing", ID: "C12345678"}},
			newChannels:      []config.Channel{},
			expectedErrCount: 1,
		},
		{
			name:            "rename a channel",
			priorChannels:   []slack.Conversation{{Name: "sig-testing", ID: "C12345678"}},
			newChannels:     []config.Channel{{Name: "sig-ponies", ID: "C12345678"}},
			expectedActions: []Action{renameChannelAction{id: "C12345678", oldName: "sig-testing", newName: "sig-ponies"}},
		},
		{
			name:             "creating an archived channel is an error",
			priorChannels:    []slack.Conversation{},
			newChannels:      []config.Channel{{Name: "sig-ponies", Archived: true}},
			expectedErrCount: 1,
		},
		{
			name:             "simultaneously create, rename, archive, and unarchive channels, while reporting an error",
			priorChannels:    []slack.Conversation{{Name: "sig-testing", ID: "C12345678"}, {Name: "sig-ponies", ID: "C11111111", IsArchived: true}, {Name: "sig-what", ID: "C22222222"}},
			newChannels:      []config.Channel{{Name: "sig-hmm", ID: "C12345678", Archived: true}, {Name: "sig-ponies"}},
			expectedActions:  []Action{renameChannelAction{id: "C12345678", oldName: "sig-testing", newName: "sig-hmm"}, archiveChannelAction{id: "C12345678", name: "sig-hmm"}, unarchiveChannelAction{id: "C11111111", name: "sig-ponies"}},
			expectedErrCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Reconciler{
				config:   config.Config{Channels: tc.newChannels},
				channels: channelState{byID: map[string]*slack.Conversation{}, byName: map[string]*slack.Conversation{}},
			}
			for _, c := range tc.priorChannels {
				c2 := c
				r.channels.byID[c.ID] = &c2
				r.channels.byName[c.Name] = &c2
			}
			actions, errs := r.reconcileChannels()
			if !reflect.DeepEqual(actions, tc.expectedActions) {
				t.Errorf("Expected actions: %#v\nActual actions: %#v", tc.expectedActions, actions)
			}
			if len(errs) != tc.expectedErrCount {
				t.Errorf("Expected %d errors, but got %d: %v", tc.expectedErrCount, len(errs), errs)
			}
		})
	}
}
