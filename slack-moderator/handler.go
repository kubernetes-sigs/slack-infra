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
	"sigs.k8s.io/slack-infra/slack"
)

type handler struct {
	client     *slack.Client
	adminToken string
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
	if interaction.Type == "message_action" && interaction.CallbackID == "report_message" {
		if isMod, err := h.userHasModerationPowers(interaction.User.ID); err == nil && isMod {
			h.handleModerateMessage(interaction, rw)
		} else {
			h.handleReportMessage(interaction, rw)
		}
	} else if interaction.Type == "dialog_submission" {
		switch interaction.CallbackID {
		case "send_report":
			h.handleReportSubmission(interaction, rw)
		case "moderate_user":
			// Spin this off because it takes longer than Slack is willing to wait for a response.
			go h.handleModerateSubmission(interaction)
		}
	}
}

func logError(rw http.ResponseWriter, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	log.Println(s)
	http.Error(rw, s, 500)
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
	State      string            `json:"state"`
}

// shortenString returns the first N slice of a string.
func shortenString(str string, n int) string {
	if len(str) <= n {
		return str
	}
	return str[:n]
}
