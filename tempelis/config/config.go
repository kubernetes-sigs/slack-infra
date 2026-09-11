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
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Config struct {
	Users           map[string]string `json:"users"`
	Channels        []Channel         `json:"channels"`
	Usergroups      []Usergroup       `json:"usergroups"`
	ChannelTemplate ChannelTemplate   `json:"channel_template,omitempty"`
	Restrictions    []Restrictions    `json:"restrictions"`
}

type Restrictions struct {
	Path             string   `json:"path"`
	Users            bool     `json:"users"`
	ChannelsString   []string `json:"channels"`
	UsergroupsString []string `json:"usergroups"`
	Template         bool     `json:"template"`

	Channels   []*regexp.Regexp
	Usergroups []*regexp.Regexp
}

type Channel struct {
	Name       string   `json:"name"`
	ID         string   `json:"id,omitempty"`
	Archived   bool     `json:"archived,omitempty"`
	Moderators []string `json:"moderators,omitempty"`
}

type Usergroup struct {
	Name        string   `json:"name,omitempty"`
	LongName    string   `json:"long_name,omitempty"`
	Members     []string `json:"members,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	Description string   `json:"description,omitempty"`
	External    bool     `json:"external,omitempty"`
}

type ChannelTemplate struct {
	Pins    []string `json:"pins,omitempty"`
	Topic   string   `json:"topic,omitempty"`
	Purpose string   `json:"purpose,omitempty"`
}

// NamesToIDs converts a list of names to a list of slack user IDs
func (c *Config) NamesToIDs(names []string) ([]string, error) {
	result := make([]string, 0, len(names))
	var missing []string
	for _, n := range names {
		if id, ok := c.Users[n]; ok {
			result = append(result, id)
		} else {
			missing = append(missing, n)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown user names: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

// maxUsergroupDescription is Slack's limit; exceeding it fails usergroups.create/update with description_too_long.
const maxUsergroupDescription = 140

// Validate checks constraints that Slack enforces at apply time but that can be
// verified offline: member names resolve, descriptions fit, and handles do not
// collide with channel names (usergroup handles and channel names share one
// namespace, so usergroups.create fails with handle_already_exists).
func (c *Config) Validate() error {
	channels := map[string]struct{}{}
	for _, ch := range c.Channels {
		channels[ch.Name] = struct{}{}
	}
	for _, g := range c.Usergroups {
		if g.External {
			continue
		}
		if _, err := c.NamesToIDs(g.Members); err != nil {
			return fmt.Errorf("usergroup %q has invalid member(s): %v", g.Name, err)
		}
		if n := utf8.RuneCountInString(g.Description); n > maxUsergroupDescription {
			return fmt.Errorf("usergroup %q description is %d characters, Slack allows at most %d", g.Name, n, maxUsergroupDescription)
		}
		if _, ok := channels[g.Name]; ok {
			return fmt.Errorf("usergroup %q has the same name as a channel; Slack rejects usergroup handles that match channel names", g.Name)
		}
	}
	return nil
}
