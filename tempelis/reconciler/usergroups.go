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
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/slack-infra/slack"
)

func (r *Reconciler) reconcileUsergroups() ([]Action, []error) {
	result := struct {
		Usergroups []slack.Subteam `json:"usergroups"`
	}{}
	if err := r.slack.CallOldMethod("usergroups.list", map[string]string{"include_users": "true", "include_disabled": "true"}, &result); err != nil {
		return nil, []error{fmt.Errorf("couldn't get usergroup list: %v", err)}
	}

	groupsByHandle := map[string]*slack.Subteam{}
	missingGroups := map[string]*slack.Subteam{}

	for _, g := range result.Usergroups {
		groupsByHandle[g.Handle] = &g
		missingGroups[g.Handle] = &g
	}

	var actions []Action
	var errors []error
	for _, g := range r.config.Usergroups {
		delete(missingGroups, g.Name)
		if o, ok := groupsByHandle[g.Name]; ok {
			if g.External {
				continue
			}
			// If we have a "deleted" group, but we found it here, it needs undeleting.
			if o.DeleteTime > 0 {
				actions = append(actions, reactivateUsergroupAction{id: o.ID, handle: o.Handle})
			}

			needsUpdate := false
			targetIDs, err := r.config.NamesToIDs(g.Members)
			if err != nil {
				errors = append(errors, fmt.Errorf("%s: %v", o.Name, err))
				continue
			}
			sort.Strings(targetIDs)
			sort.Strings(o.Users)

			targetChannels, err := r.channels.namesToIDs(g.Channels)
			if err != nil {
				errors = append(errors, fmt.Errorf("%s: %v", o.Name, err))
				continue
			}
			sort.Strings(targetChannels)
			sort.Strings(o.Prefs.Channels)

			needsUpdate = needsUpdate || o.Name != g.LongName
			needsUpdate = needsUpdate || o.Description != g.Description
			needsUpdate = needsUpdate || !stringSlicesEqual(targetChannels, o.Prefs.Channels)

			if needsUpdate {
				actions = append(actions, updateUsergroupAction{id: o.ID, description: g.Description, name: g.LongName, channelNames: g.Channels})
			}

			if !stringSlicesEqual(o.Users, targetIDs) {
				actions = append(actions, updateUsergroupMembersAction{id: o.ID, users: targetIDs})
			}
		} else {
			targetIDs, err := r.config.NamesToIDs(g.Members)
			if err != nil {
				errors = append(errors, fmt.Errorf("%s: %v", o.Name, err))
				continue
			}
			actions = append(actions, updateUsergroupAction{id: o.ID, description: g.Description, name: g.LongName, channelNames: g.Channels, create: true}, updateUsergroupMembersAction{id: o.ID, users: targetIDs})
		}
	}

	for _, o := range missingGroups {
		actions = append(actions, deactivateUsergroupAction{id: o.ID, handle: o.Handle})
	}

	return actions, errors
}

func stringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		for j := range b {
			if a[i] != b[j] {
				return false
			}
		}
	}
	return true
}

type deactivateUsergroupAction struct {
	id     string
	handle string
}

func (a deactivateUsergroupAction) Describe() string {
	return fmt.Sprintf("Deactivate usergroup %s (%s)", a.handle, a.id)
}

func (a deactivateUsergroupAction) Perform(reconciler *Reconciler) error {
	if err := reconciler.slack.CallMethod("usergroups.disable", map[string]string{"usergroup": a.id}, nil); err != nil {
		return fmt.Errorf("failed to disable usergroup %s (%s): %v", a.handle, a.id, err)
	}
	return nil
}

type reactivateUsergroupAction struct {
	id     string
	handle string
}

func (a reactivateUsergroupAction) Describe() string {
	return fmt.Sprintf("Reactivate usergroup: %s", a.handle)
}

func (a reactivateUsergroupAction) Perform(reconciler *Reconciler) error {
	if err := reconciler.slack.CallMethod("usergroups.enable", map[string]string{"usergroup": a.id}, nil); err != nil {
		return fmt.Errorf("failed to reactivate usergroup %s (%s): %v", a.handle, a.id, err)
	}
	return nil
}

type updateUsergroupAction struct {
	id           string
	description  string
	name         string
	channelNames []string
	create       bool
}

func (a updateUsergroupAction) Describe() string {
	return fmt.Sprintf("Unarchive channel: %s", a.name)
}

func (a updateUsergroupAction) Perform(reconciler *Reconciler) error {
	channelIDs, err := reconciler.channels.namesToIDs(a.channelNames)
	if err != nil {
		return fmt.Errorf("couldn't find channel IDs for usergroup %s: %v", a.name, err)
	}
	for _, c := range channelIDs {
		if c == "" {
			return fmt.Errorf("unexpected empty channel ID when updating usergroup %s", a.name)
		}
	}

	req := map[string]string{
		"usergroup":   a.id,
		"channels":    strings.Join(channelIDs, ","),
		"description": a.description,
		"name":        a.name,
	}

	action := "usergroups.update"
	if a.create {
		action = "usergroups.create"
	}

	if err := reconciler.slack.CallMethod(action, req, nil); err != nil {
		return fmt.Errorf("failed to update usergroup %s (%s): %v", a.name, a.id, err)
	}
	return nil
}

type updateUsergroupMembersAction struct {
	id    string
	users []string
}

func (a updateUsergroupMembersAction) Describe() string {
	return fmt.Sprintf("Set members of usergroup %s to %v", a.id, a.users)
}

func (a updateUsergroupMembersAction) Perform(reconciler *Reconciler) error {
	if err := reconciler.slack.CallMethod("usergroups.users.update", map[string]string{"usergroup": a.id, "users": strings.Join(a.users, ",")}, nil); err != nil {
		return fmt.Errorf("failed to update members of usergroup %s: %v", a.id, err)
	}
	return nil
}
