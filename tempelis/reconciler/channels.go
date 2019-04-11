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

	"sigs.k8s.io/slack-infra/slack"
)

func (r *Reconciler) reconcileChannels() ([]Action, []error) {
	missingChannels := map[string]*slack.Conversation{}

	var actions []Action
	var errors []error

	for _, c := range r.channels.byName {
		missingChannels[c.Name] = c
	}

	for _, c := range r.config.Channels {
		if c.ID != "" {
			if o, ok := r.channels.byID[c.ID]; ok {
				if o.Name != c.Name {
					oldName := o.Name
					if err := r.channels.rename(oldName, c.Name); err != nil {
						errors = append(errors, err)
					} else {
						actions = append(actions, renameChannelAction{id: o.ID, oldName: oldName, newName: c.Name})
					}
					delete(missingChannels, oldName)
				}
			} else {
				errors = append(errors, fmt.Errorf("channel ID %s (for channel named %s) specified, but not known to Slack", c.ID, c.Name))
			}
		}
		if o, ok := r.channels.byName[c.Name]; ok {
			if c.Archived && !o.IsArchived {
				actions = append(actions, archiveChannelAction{id: o.ID, name: o.Name})
			} else if !c.Archived && o.IsArchived {
				actions = append(actions, unarchiveChannelAction{id: o.ID, name: o.Name})
			}
			delete(missingChannels, o.Name)
		} else {
			actions = append(actions, createChannelAction{name: c.Name})
		}
	}

	for _, o := range missingChannels {
		errors = append(errors, fmt.Errorf("channel %s (%s) not referenced in config", o.Name, o.ID))
	}

	return actions, errors
}

type createChannelAction struct {
	name string
}

func (a createChannelAction) Describe() string {
	return fmt.Sprintf("Create new channel: %s", a.name)
}

func (a createChannelAction) Perform(reconciler *Reconciler) error {
	ret := struct {
		Channel slack.Conversation `json:"channel"`
	}{}
	if err := reconciler.slack.CallMethod("conversations.create", map[string]string{"name": a.name}, &ret); err != nil {
		return fmt.Errorf("failed to create channel: %v", err)
	}
	c := ret.Channel
	reconciler.channels.byName[c.Name] = &c
	reconciler.channels.byID[c.Name] = &c
	t := &reconciler.config.ChannelTemplate
	if t.Topic != "" {
		if err := reconciler.slack.CallMethod("conversations.setTopic", map[string]string{"channel": c.ID, "topic": t.Topic}, nil); err != nil {
			return fmt.Errorf("failed to set topic of channel %s to %q: %v", c.Name, t.Topic, err)
		}
	}
	if t.Purpose != "" {
		if err := reconciler.slack.CallMethod("conversations.setPurpose", map[string]interface{}{"channel": c.ID, "purpose": t.Purpose}, nil); err != nil {
			return fmt.Errorf("failed to set purpose of channel %s to %q: %v", c.Name, c.Topic, err)
		}
	}
	for _, p := range t.Pins {
		message := struct {
			Channel   string `json:"channel"`
			Text      string `json:"text"`
			AsUser    bool   `json:"as_user"`
			LinkNames bool   `json:"link_names"`
		}{
			Channel:   c.ID,
			Text:      p,
			AsUser:    false,
			LinkNames: true,
		}
		r := struct {
			TS string `json:"ts"`
		}{}
		if err := reconciler.slack.CallMethod("chat.postMessage", message, &r); err != nil {
			return fmt.Errorf("failed to send message: %v", err)
		}
		if err := reconciler.slack.CallMethod("pins.add", map[string]string{"channel": c.ID, "timestamp": r.TS}, nil); err != nil {
			return fmt.Errorf("failed to pin message %s in %s: %v", r.TS, c.Name, err)
		}
	}
	return nil
}

type unarchiveChannelAction struct {
	id   string
	name string
}

func (a unarchiveChannelAction) Describe() string {
	return fmt.Sprintf("Unarchive channel: %s", a.name)
}

func (a unarchiveChannelAction) Perform(reconciler *Reconciler) error {
	if err := reconciler.slack.CallMethod("conversations.unarchive", map[string]string{"channel": a.id}, nil); err != nil {
		return fmt.Errorf("failed to unarchive channel %s (%s): %v", a.id, a.name, err)
	}
	return nil
}

type archiveChannelAction struct {
	id   string
	name string
}

func (a archiveChannelAction) Describe() string {
	return fmt.Sprintf("Archive channel: %s", a.name)
}

func (a archiveChannelAction) Perform(reconciler *Reconciler) error {
	if err := reconciler.slack.CallMethod("conversations.archive", map[string]string{"channel": a.id}, nil); err != nil {
		return fmt.Errorf("failed to archive channel %s (%s): %v", a.name, a.id, err)
	}
	return nil
}

type renameChannelAction struct {
	id      string
	oldName string
	newName string
}

func (a renameChannelAction) Describe() string {
	return fmt.Sprintf("Rename channel %s from %s to %s", a.id, a.oldName, a.newName)
}

func (a renameChannelAction) Perform(reconciler *Reconciler) error {
	if err := reconciler.slack.CallMethod("conversations.rename", map[string]string{"channel": a.id, "name": a.newName}, nil); err != nil {
		return fmt.Errorf("failed to rename channel %s (%s) to %s: %v", a.oldName, a.id, a.newName, err)
	}
	return nil
}
