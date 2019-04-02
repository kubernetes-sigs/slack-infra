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
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

func (h *handler) handleModerateMessage(interaction slackInteraction, rw http.ResponseWriter) {
	targetUser, err := h.getDisplayName(interaction.Message.User)
	if err != nil {
		targetUser = "<error>"
	}
	deactivateElement := slack.SelectElement{
		Name:  "deactivate",
		Label: fmt.Sprintf("Deactivate %s (%s)?", shortenString(slack.EscapeMessage(targetUser), 24), interaction.Message.User),
		Options: []slack.SelectOption{
			{
				Label: "No",
				Value: "no",
			},
			{
				Label: "Yes",
				Value: "yes",
			},
		},
		Value: "no",
	}
	removeContentElement := slack.SelectElement{
		Name:  "remove_content",
		Label: fmt.Sprintf("How much content would you like to remove?"),
		Options: []slack.SelectOption{
			{
				Label: "None",
				Value: "none",
			},
			{
				Label: "10 minutes",
				Value: "10m",
			},
			{
				Label: "1 hour",
				Value: "1h",
			},
			{
				Label: "6 hours",
				Value: "6h",
			},
			{
				Label: "12 hours",
				Value: "12h",
			},
			{
				Label: "24 hours",
				Value: "24h",
			},
			{
				Label: "48 hours",
				Value: "48h",
			},
		},
		Value: "10m",
	}
	dialog := slack.DialogWrapper{
		TriggerID: interaction.TriggerID,
		Dialog: slack.Dialog{
			CallbackID:     "moderate_user",
			NotifyOnCancel: false,
			Title:          "Moderate User",
			Elements:       []interface{}{deactivateElement, removeContentElement},
			State:          interaction.Message.User,
		},
	}
	if err := h.client.CallMethod("dialog.open", dialog, nil); err != nil {
		logError(rw, "Failed to call dialog.open: %v", err)
		return
	}
}

func (h *handler) handleModerateSubmission(interaction slackInteraction) {
	isMod, err := h.userHasModerationPowers(interaction.User.ID)
	if err != nil || !isMod {
		log.Printf("User %s (%s) does not seem to be a mod: %v\n", interaction.User.ID, interaction.User.Name, err)
		return
	}
	var messages []string
	targetUser := interaction.State
	targetDisplayName, err := h.getDisplayName(targetUser)
	if err != nil {
		targetDisplayName = "<unknown>"
	}

	quickResponse := map[string]interface{}{
		"text":             "Please wait...",
		"response_type":    "ephemeral",
		"replace_original": false,
	}
	if h.client.CallMethod(interaction.ResponseURL, quickResponse, nil) != nil {
		log.Printf("Failed to send quick response: %v.\n", err)
	}

	modMessage := fmt.Sprintf("<@%s> triggered moderation on <@%s>. Deactivate: %s, remove content: %s", interaction.User.ID, targetUser, interaction.Submission["deactivate"], interaction.Submission["remove_content"])
	if h.client.CallMethod(h.client.Config.WebhookURL, map[string]string{"text": modMessage}, nil) != nil {
		log.Printf("Failed to send quick response: %v.\n", err)
	}

	if interaction.Submission["deactivate"] == "yes" {
		if err := h.deactivateUser(interaction, targetUser); err != nil {
			messages = append(messages, fmt.Sprintf("Failed to deactivate user %s (%s): %v", targetUser, targetDisplayName, err))
		} else {
			messages = append(messages, fmt.Sprintf("Successfully deactivated user %s (%s)", targetUser, targetDisplayName))
		}
	}
	if remove, ok := interaction.Submission["remove_content"]; ok && remove != "none" {
		duration, err := time.ParseDuration(remove)
		if err != nil {
			messages = append(messages, fmt.Sprintf("Failed to parse removal duration, and therefore could not remove any content: %v", err))
		}
		removedFiles, remainingFiles, removedMessages, remainingMessages, err := h.removeUserContent(interaction, duration, targetUser)

		if err != nil {
			messages = append(messages, fmt.Sprintf("Failed to remove any content: %v", err))
		}

		if remainingFiles == 0 && remainingMessages == 0 {
			messages = append(messages, fmt.Sprintf("Successfully removed %d messages and %d files", removedMessages, removedFiles))
		} else {
			messages = append(messages, fmt.Sprintf("Couldn't remove all content. Removed %d messages and %d files, but there are %d messages and %d files left.", removedMessages, removedFiles, remainingMessages, remainingFiles))
		}
	}

	if len(messages) == 0 {
		messages = append(messages, "Did nothing.")
	}

	response := map[string]interface{}{
		"text":             strings.Join(messages, "\n"),
		"response_type":    "ephemeral",
		"replace_original": true,
	}

	if h.client.CallMethod(interaction.ResponseURL, response, nil) != nil {
		log.Printf("Failed to send response: %v.\n", err)
	}
	if h.client.CallMethod(h.client.Config.WebhookURL, map[string]string{"text": strings.Join(messages, "\n")}, nil) != nil {
		log.Printf("Failed to send quick response: %v.\n", err)
	}
}
