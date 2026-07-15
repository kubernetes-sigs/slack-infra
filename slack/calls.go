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

package slack

import (
	"fmt"
	"strings"
	"time"
)

type ConversationType string

const (
	ConversationTypePublicChannel  ConversationType = "public_channel"
	ConversationTypePrivateChannel ConversationType = "private_channel"
	ConversationTypeIM             ConversationType = "im"
	ConversationTypeMPIM           ConversationType = "mpip"
)

func (c *Client) GetConversations(types []ConversationType) ([]Conversation, error) {
	t := make([]string, 0, len(types))
	for _, v := range types {
		t = append(t, string(v))
	}

	var conversations []Conversation
	cursor := ""
	for {
		args := map[string]string{
			"limit": "100",
			"types": strings.Join(t, ","),
		}
		if cursor != "" {
			args["cursor"] = cursor
		}

		ret := struct {
			Channels []Conversation `json:"channels"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}{}

		for {
			if err := c.CallOldMethod("conversations.list", args, &ret); err != nil {
				switch e := err.(type) {
				case ErrRateLimit:
					time.Sleep(e.Wait)
					continue
				default:
					return nil, fmt.Errorf("failed to list conversations: %v", err)
				}
			}
			break
		}

		conversations = append(conversations, ret.Channels...)
		if ret.Metadata.NextCursor == "" {
			break
		}
		cursor = ret.Metadata.NextCursor
	}
	return conversations, nil
}

func (c *Client) GetPublicChannels() ([]Conversation, error) {
	return c.GetConversations([]ConversationType{ConversationTypePublicChannel})
}

// GetConversationMembers returns the IDs of every member of a conversation,
// following cursor pagination.
func (c *Client) GetConversationMembers(channelID string) ([]string, error) {
	var members []string
	cursor := ""
	for {
		args := map[string]string{
			"channel": channelID,
			"limit":   "1000",
		}
		if cursor != "" {
			args["cursor"] = cursor
		}

		ret := struct {
			Members  []string `json:"members"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}{}

		for {
			if err := c.CallOldMethod("conversations.members", args, &ret); err != nil {
				switch e := err.(type) {
				case ErrRateLimit:
					time.Sleep(e.Wait)
					continue
				default:
					return nil, fmt.Errorf("failed to list conversation members: %v", err)
				}
			}
			break
		}

		members = append(members, ret.Members...)
		if ret.Metadata.NextCursor == "" {
			break
		}
		cursor = ret.Metadata.NextCursor
	}
	return members, nil
}

// KickFromConversation removes a user from a conversation. On failure it returns
// the underlying error, which for Slack API errors is an ErrSlack whose Type
// holds the Slack error code (e.g. cant_kick_from_general, not_in_channel,
// restricted_action), so callers can decide whether to skip or abort.
func (c *Client) KickFromConversation(channelID, userID string) error {
	args := map[string]string{
		"channel": channelID,
		"user":    userID,
	}
	for {
		if err := c.CallOldMethod("conversations.kick", args, nil); err != nil {
			switch e := err.(type) {
			case ErrRateLimit:
				time.Sleep(e.Wait)
				continue
			default:
				return err
			}
		}
		return nil
	}
}
