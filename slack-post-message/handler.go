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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"go4.org/sort"
	"sigs.k8s.io/slack-infra/slack"
)

type handler struct {
	client     *slack.Client
	userGroups []string
}

func logError(rw http.ResponseWriter, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	log.Println(s)
	http.Error(rw, s, 500)
}

// ServeHTTP handles Slack webhook requests.
func (h *handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logError(rw, "Failed to read incoming request body: %v", err)
		return
	}
	if err := h.client.VerifySignature(body, r.Header); err != nil {
		logError(rw, "Failed validation: %v", err)
		return
	}
	f, err := url.ParseQuery(string(body))
	if err != nil {
		logError(rw, "Failed to parse incoming content: %v", err)
		return
	}
	content := f.Get("payload")
	if content == "" {
		logError(rw, "Payload was blank.")
		return
	}
	interaction := slackInteraction{}
	if err := json.Unmarshal([]byte(content), &interaction); err != nil {
		logError(rw, "Failed to unmarshal payload: %v", err)
		return
	}
	if h.handlePermissionCheck(interaction, rw) {
		if interaction.Type == "shortcut" && interaction.CallbackID == "write_message" {
			h.handleWriteMessage(interaction, rw)
		} else if interaction.Type == "view_submission" && interaction.View.CallbackID == "post_message" {
			h.handlePostMessage(interaction, rw)
		}
	} else {
		h.handleNotInGroupError(interaction, rw)
	}
}

// Checks if the user using the bot has permissions
func (h *handler) handlePermissionCheck(interaction slackInteraction, rw http.ResponseWriter) bool {
	for _, value := range h.userGroups {
		result := struct {
			Ok    bool     `json:"ok"`
			Users []string `json:"users"`
		}{}
		args := map[string]string{
			"usergroup": value,
		}
		if err := h.client.CallOldMethod("usergroups.users.list", args, &result); err != nil {
			logError(rw, "Failed to call usergroups.users.list: %v", err)
			return false
		}
		for _, user := range result.Users {
			if user == interaction.User.ID {
				return true
			}
		}
	}
	return false
}

// Shows a error message, if user using the bot doesn't have permissions
func (h *handler) handleNotInGroupError(interaction slackInteraction, rw http.ResponseWriter) {
	sectionBlock := SectionBlock{
		Text: TextObject{
			Type: "plain_text",
			Text: "Only users part of the configured usergroup(s) are authorized to use this bot. Please check with slack admins for more info.",
		},
	}
	view := View{
		Type: "modal",
		Title: TextObject{
			Type: "plain_text",
			Text: "Not authorized",
		},
		Close: TextObject{
			Type: "plain_text",
			Text: "Ok",
		},
		Blocks: []interface{}{sectionBlock},
	}
	args := map[string]interface{}{
		"trigger_id": interaction.TriggerID,
		"view":       view,
	}
	if err := h.client.CallMethod("views.open", args, nil); err != nil {
		logError(rw, "Failed to call views.open: %v", err)
		return
	}
}

// Opens a slack Modal that allows users to choose channels and compose a message
func (h *handler) handleWriteMessage(interaction slackInteraction, rw http.ResponseWriter) {
	channelSelect := MultiSelectChannelElement{
		ActionID: "channel-input",
		Placeholder: TextObject{
			Type: "plain_text",
			Text: "Select channels",
		},
	}
	messageInput := PlainTextInputElement{
		ActionID:  "channel-block",
		Multiline: true,
		Placeholder: TextObject{
			Type: "plain_text",
			Text: "Write a message",
		},
	}
	sectionBlock := SectionBlock{
		Text: TextObject{
			Type: "plain_text",
			Text: "Use this form to post message in a slack channel.",
		},
	}
	dividerBlock := DividerBlock{}
	inputBlock1 := InputBlock{
		BlockID: "message-input",
		Label: TextObject{
			Type: "plain_text",
			Text: "Pick channel(s) from the list",
		},
		Hint: TextObject{
			Type: "plain_text",
			Text: "Pick a channel",
		},
		Element: &channelSelect,
	}
	inputBlock2 := InputBlock{
		BlockID: "message-block",
		Label: TextObject{
			Type: "plain_text",
			Text: "Message",
		},
		Hint: TextObject{
			Type: "plain_text",
			Text: "Enter a message",
		},
		Element: &messageInput,
	}
	view := View{
		Type:       "modal",
		CallbackID: "post_message",
		Title: TextObject{
			Type: "plain_text",
			Text: "Post Message",
		},
		Submit: TextObject{
			Type: "plain_text",
			Text: "Submit",
		},
		Close: TextObject{
			Type: "plain_text",
			Text: "Cancel",
		},
		Blocks: []interface{}{sectionBlock, dividerBlock, inputBlock1, inputBlock2},
	}
	args := map[string]interface{}{
		"trigger_id": interaction.TriggerID,
		"view":       view,
	}
	if err := h.client.CallMethod("views.open", args, nil); err != nil {
		logError(rw, "Failed to call views.open: %v", err)
		return
	}
}

// Posts messages in the channels chosen by the user
func (h *handler) handlePostMessage(interaction slackInteraction, rw http.ResponseWriter) {
	channels := interaction.View.State.Values.Block1.Element.SelectedChannels
	message := interaction.View.State.Values.Block2.Element.Value
	result := struct {
		Ok       bool `json:"ok"`
		Channels []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			IsChannel bool   `json:"is_channel"`
		} `json:"channels"`
	}{}
	if err := h.client.CallMethod("users.conversations", nil, &result); err != nil {
		logError(rw, "Failed to send users.conversations: %v.", err)
	}
	channelsJoined := []string{}
	for i := 0; i < len(result.Channels); i++ {
		if result.Channels[i].IsChannel {
			channelsJoined = append(channelsJoined, result.Channels[i].ID)
		}
	}
	sort.Strings(channelsJoined)
	for i := 0; i < len(channels); i++ {
		index := sort.SearchStrings(channelsJoined, channels[i])
		if index >= len(channelsJoined) || channelsJoined[index] != channels[i] {
			h.handleNotInChannelError(interaction, rw, channels[i])
			return
		}
	}
	for i := 0; i < len(channels); i++ {
		args := map[string]interface{}{
			"channel": channels[i],
			"text":    message,
		}
		if err := h.client.CallMethod("chat.postMessage", args, nil); err != nil {
			logError(rw, "Failed to send chat.postMessage: %v.", err)
			return
		}
	}
}

// If the bot is not added in a channel, shows an error. Only when bot is added in all the channel chosen the message will be posted
func (h *handler) handleNotInChannelError(interaction slackInteraction, rw http.ResponseWriter, channel string) {
	result := struct {
		Ok      bool `json:"ok"`
		Channel struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			IsChannel bool   `json:"is_channel"`
		} `json:"channel"`
	}{}
	args := map[string]string{
		"channel": channel,
	}
	if err := h.client.CallOldMethod("conversations.info", args, &result); err != nil {
		logError(rw, "Failed to send conversations.info: %v.", err)
	}
	quickResponse := map[string]interface{}{
		"response_action": "errors",
		"errors": map[string]interface{}{
			"message-input": "Please add bot to the channel '" + result.Channel.Name + "' before posting a message",
		},
	}
	json, err := json.Marshal(quickResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(json)
}

type slackInteraction struct {
	Token       string `json:"token"`
	CallbackID  string `json:"callback_id"`
	Type        string `json:"type"`
	TriggerID   string `json:"trigger_id"`
	ResponseURL string `json:"response_url"`
	Team        struct {
		ID     string `json:"id"`
		Domain string `json:"string"`
	} `json:"team"`
	Channel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channel"`
	User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"user"`
	Message struct {
		Type      string `json:"type"`
		User      string `json:"user"`
		Timestamp string `json:"ts"`
		Text      string `json:"text"`
	}
	Submission map[string]string `json:"submission"`
	View       struct {
		ViewID     string `json:"id"`
		CallbackID string `json:"callback_id"`
		State      struct {
			Values struct {
				Block1 struct {
					Element struct {
						Type             string   `json:"type"`
						SelectedChannels []string `json:"selected_channels"`
					} `json:"channel-input"`
				} `json:"message-input"`
				Block2 struct {
					Element struct {
						Type  string `json:"type"`
						Value string `json:"value"`
					} `json:"channel-block"`
				} `json:"message-block"`
			} `json:"values"`
		} `json:"state"`
	} `json:"view"`
}
