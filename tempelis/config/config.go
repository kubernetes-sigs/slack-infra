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
	Deny             bool     `json:"deny"`
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
