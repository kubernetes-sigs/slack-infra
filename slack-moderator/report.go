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
	"log"
	"net/http"
	"strconv"

	"sigs.k8s.io/slack-infra/slack"
)

func (h *handler) handleReportMessage(interaction slackInteraction, rw http.ResponseWriter) {
	textArea := slack.TextArea{
		Name:  "message",
		Label: "Why are you reporting this message?",
		Hint:  "Moderators will see whatever you write here, along with the message being reported.",
	}
	selectElement := slack.SelectElement{
		Name:  "anonymous",
		Label: "Would you like to report anonymously?",
		Options: []slack.SelectOption{
			{
				Label: "No, report with my username",
				Value: "no",
			},
			{
				Label: "Yes, report anonymously",
				Value: "yes",
			},
		},
		Value: "no",
	}
	var elements []interface{}
	if interaction.Channel.Name == "directmessage" {
		elements = []interface{}{textArea}
	} else {
		elements = []interface{}{textArea, selectElement}
	}
	state, err := json.Marshal(dialogState{
		Sender:  interaction.Message.User,
		TS:      interaction.Message.Timestamp,
		Content: shortenString(interaction.Message.Text, 2900),
	})
	if err != nil {
		logError(rw, "Failed to serialise state for dialog: %v", err)
		return
	}
	dialog := slack.DialogWrapper{
		TriggerID: interaction.TriggerID,
		Dialog: slack.Dialog{
			CallbackID:     "send_report",
			NotifyOnCancel: false,
			Title:          "Report Message",
			Elements:       elements,
			State:          string(state),
		},
	}
	if err := h.client.CallMethod("dialog.open", dialog, nil); err != nil {
		logError(rw, "Failed to call dialog.open: %v", err)
		return
	}
}

func (h *handler) handleReportSubmission(interaction slackInteraction, rw http.ResponseWriter) {
	anonymous := interaction.Submission["anonymous"] == "yes"
	message := interaction.Submission["message"]
	state := dialogState{}
	if err := json.Unmarshal([]byte(interaction.State), &state); err != nil {
		logError(rw, "Failed to parse provided state: %v.", err)
		return
	}

	// Construct summary string
	var who string
	if anonymous {
		who = "An anonymous user"
	} else {
		who = fmt.Sprintf("<@%s|%s>", interaction.User.ID, interaction.User.Name)
	}

	var where string
	if interaction.Channel.Name == "directmessage" {
		where = "a direct message"
	} else {
		where = fmt.Sprintf("<#%s|%s>", interaction.Channel.ID, interaction.Channel.Name)
	}

	summary := fmt.Sprintf("%s *reported a message* in %s:", who, where)

	// Figure out a timestamp from the combined timestamp/message ID
	ts, err := strconv.ParseFloat(state.TS, 64)
	if err != nil {
		logError(rw, "Failed to parse provided timestamp: %v.", err)
		return
	}

	messageLink := "message they reported"
	if interaction.Channel.Name != "directmessage" {
		permalink, err := h.getPermalink(interaction.Channel.ID, state.TS)
		if err != nil {
			log.Printf("Failed to get a permalink: %v.", err)
		} else {
			messageLink = fmt.Sprintf("<%s|message they reported>", permalink)
		}
	}

	var author string
	if senderName, err := h.getDisplayName(state.Sender); err == nil {
		author = fmt.Sprintf("<@%s|%s>", state.Sender, senderName)
	} else {
		author = fmt.Sprintf("<@%s>", state.Sender)
		log.Printf("Failed to look up sender: %v", err)
	}

	report := map[string]interface{}{
		"text": summary,
		"attachments": []map[string]interface{}{
			{
				"pretext":   "They said:",
				"text":      message,
				"mrkdwn_in": []string{"text"},
				"fallback":  "They said: " + message,
			},
			{
				"pretext":     fmt.Sprintf("The %s was:", messageLink),
				"author_name": author,
				"text":        state.Content,
				"ts":          ts,
				"mrkdwn_in":   []string{"text", "pretext", "author_name"},
				"fallback":    fmt.Sprintf("The message they reported was: %s", state.Content),
			},
		},
	}
	if err := h.client.CallMethod(h.client.Config.WebhookURL, report, nil); err != nil {
		logError(rw, "Failed to send report: %v.", err)
		return
	}

	response := map[string]interface{}{
		"text":             "Thank you! Your report has been submitted.",
		"response_type":    "ephemeral",
		"replace_original": false,
	}

	if h.client.CallMethod(interaction.ResponseURL, response, nil) != nil {
		logError(rw, "Failed to send response: %v.", err)
		return
	}
}

func (h *handler) getPermalink(channel string, ts string) (string, error) {
	permalink := struct {
		Channel   string `json:"string"`
		Permalink string `json:"permalink"`
	}{}

	args := map[string]string{
		"channel":    channel,
		"message_ts": ts,
	}

	if err := h.client.CallOldMethod("chat.getPermalink", args, &permalink); err != nil {
		return "", fmt.Errorf("failed get permalink: %v", err)
	}
	return permalink.Permalink, nil
}

// The JSON strings here are short because we can only put a limited amount of information in
// the dialog state.
type dialogState struct {
	Sender  string `json:"s"`
	TS      string `json:"t"`
	Content string `json:"c"`
}
