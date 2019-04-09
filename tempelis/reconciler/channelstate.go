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
	"strings"
)

type channelState struct {
	byName map[string]*slack.Conversation
	byID   map[string]*slack.Conversation
}

func (c *channelState) init(s *slack.Client) error {
	channels, err := s.GetPublicChannels()
	if err != nil {
		return err
	}
	for _, ch := range channels {
		c.byName[ch.Name] = &ch
		c.byID[ch.ID] = &ch
	}
	return nil
}

func (c *channelState) rename(old, new string) error {
	if _, ok := c.byName[new]; ok {
		return fmt.Errorf("can't rename %s to %s: name already used", old, new)
	}
	c.byName[old].Name = new
	c.byName[new] = c.byName[old]
	delete(c.byName, old)
	return nil
}

func (c *channelState) create(name string) error {
	if _, ok := c.byName[name]; ok {
		return fmt.Errorf("can't create channel %s: name already used", name)
	}
	conversation := slack.Conversation{
		Name:      name,
		IsChannel: true,
	}
	c.byName[name] = &conversation
	return nil
}

func (c *channelState) namesToIDs(names []string) ([]string, error) {
	result := make([]string, 0, len(names))
	var missing []string
	for _, n := range names {
		if r, ok := c.byName[n]; ok {
			result = append(result, r.ID)
		} else {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("couldn't find channel IDs: %s", strings.Join(missing, ", "))
	}
	return result, nil
}
