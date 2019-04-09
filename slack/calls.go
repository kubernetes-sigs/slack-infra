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
