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
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

const (
	workflowNotice = `Hi there! Your message in <#%s> was removed because that channel only accepts posts submitted via the official workflow.

To post a job listing or resume, please use the workflow shortcut (:zap:) in that channel.

If you have questions, reach out in #slack-admins.`

	maxRetries = 5
)

type handler struct {
	client      *slack.Client
	messagePath string
}

func logError(rw http.ResponseWriter, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	log.Println(s)
	http.Error(rw, s, 500)
}

type handlerFunc func(body []byte) ([]byte, error)

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
	response, err := h.handleMessage(body)
	if err != nil {
		logError(rw, "Failed to handle message: %v", err)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(response)
}

func (h *handler) handleMessage(body []byte) ([]byte, error) {
	t := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, err
	}

	messageMapping := map[string]handlerFunc{
		"url_verification": h.handleURLVerification,
		"event_callback":   h.handleEvent,
	}

	fn, ok := messageMapping[t.Type]
	if !ok {
		return nil, fmt.Errorf("unknown event type %q", t.Type)
	}
	output, err := fn(body)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", t.Type, err)
	}
	return output, nil
}

func (h *handler) handleURLVerification(body []byte) ([]byte, error) {
	request := struct {
		Challenge string `json:"challenge"`
	}{}
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("error parsing request: %v", err)
	}
	response := map[string]string{"challenge": request.Challenge}
	return json.Marshal(response)
}

func (h *handler) handleEvent(body []byte) ([]byte, error) {
	t := struct {
		Event struct {
			Type string `json:"type"`
		} `json:"event"`
	}{}
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	switch t.Event.Type {
	case "team_join":
		event := struct {
			Event struct {
				User slack.User `json:"user"`
			} `json:"event"`
		}{}
		if err := json.Unmarshal(body, &event); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %v", err)
		}
		if err := h.sendWelcome(event.Event.User.ID); err != nil {
			return nil, fmt.Errorf("failed to send welcome: %v", err)
		}
	case "message":
		event := struct {
			Event struct {
				SubType string `json:"subtype"`
				User    string `json:"user"`
				Channel string `json:"channel"`
				Ts      string `json:"ts"`
				BotID   string `json:"bot_id"`
			} `json:"event"`
		}{}
		if err := json.Unmarshal(body, &event); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %v", err)
		}
		if event.Event.BotID != "" || event.Event.SubType == "bot_message" {
			return []byte{}, nil
		}
		if h.isGuardedChannel(event.Event.Channel) && event.Event.User != "" {
			if err := h.enforceWorkflowOnly(event.Event.Channel, event.Event.User, event.Event.Ts); err != nil {
				return nil, fmt.Errorf("failed to enforce channel rules: %v", err)
			}
		}
	}

	return []byte{}, nil
}

func (h *handler) isGuardedChannel(channel string) bool {
	for _, c := range h.client.Config.GuardedChannels {
		if c == channel {
			return true
		}
	}
	return false
}

func (h *handler) getUserInfo(id string) (slack.User, error) {
	user := struct {
		User slack.User `json:"user"`
	}{}
	if err := h.client.CallOldMethod("users.info", map[string]string{"user": id}, &user); err != nil {
		return slack.User{}, fmt.Errorf("failed get user: %v", err)
	}
	return user.User, nil
}

func (h *handler) userIsAdmin(id string) (bool, error) {
	user, err := h.getUserInfo(id)
	if err != nil {
		log.Printf("Failed to look up admin status: %v\n", err)
		return false, err
	}
	return user.IsAdmin || user.IsOwner || user.IsPrimaryOwner, nil
}

func (h *handler) enforceWorkflowOnly(channel, userID, ts string) error {
	admin, err := h.userIsAdmin(userID)
	if err != nil {
		log.Printf("Could not verify admin status for user %s, allowing post: %v\n", userID, err)
		return nil
	}
	if admin {
		return nil
	}

	log.Printf("Removing direct post from user %s in channel %s\n", userID, channel)

	adminClient := slack.New(slack.Config{
		AccessToken:   h.client.Config.AdminToken,
		SigningSecret: h.client.Config.SigningSecret,
	})

	req := map[string]interface{}{
		"channel": channel,
		"ts":      ts,
		"as_user": true,
	}
	for i := 0; i < maxRetries; i++ {
		err := adminClient.CallMethod("chat.delete", req, nil)
		if err == nil {
			break
		}
		switch e := err.(type) {
		case slack.ErrRateLimit:
			if i == maxRetries-1 {
				return fmt.Errorf("rate limit exceeded after %d retries: %v", maxRetries, err)
			}
			log.Printf("Slack is rate limiting us, trying again in %s...\n", e.Wait)
			time.Sleep(e.Wait)
		case slack.ErrSlack:
			if e.Type == "message_not_found" {
				log.Printf("Message to delete not found, probably already deleted.\n")
				return nil
			}
			return err
		default:
			return err
		}
	}

	if err := h.notifyUser(userID, channel); err != nil {
		log.Printf("Deleted message but failed to DM user %s: %v\n", userID, err)
	}
	return nil
}

func (h *handler) notifyUser(userID, channelID string) error {
	response := struct {
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}{}
	if err := h.client.CallMethod("conversations.open", map[string]string{"users": userID}, &response); err != nil {
		return fmt.Errorf("couldn't open a conversation channel: %v", err)
	}
	message := struct {
		Channel   string `json:"channel"`
		Text      string `json:"text"`
		AsUser    bool   `json:"as_user"`
		LinkNames bool   `json:"link_names"`
	}{
		Channel:   response.Channel.ID,
		Text:      fmt.Sprintf(workflowNotice, channelID),
		AsUser:    true,
		LinkNames: true,
	}
	if err := h.client.CallMethod("chat.postMessage", message, nil); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	return nil
}

func (h *handler) sendWelcome(uid string) error {
	welcome, err := h.getWelcome()
	if err != nil {
		return fmt.Errorf("couldn't get welcome: %v", err)
	}

	response := struct {
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}{}

	if err := h.client.CallMethod("conversations.open", map[string]string{"users": uid}, &response); err != nil {
		return fmt.Errorf("couldn't open a conversation channel: %v", err)
	}
	channel := response.Channel.ID

	message := struct {
		Channel   string `json:"channel"`
		Text      string `json:"text"`
		AsUser    bool   `json:"as_user"`
		LinkNames bool   `json:"link_names"`
	}{
		Channel:   channel,
		Text:      welcome,
		AsUser:    true, // Send messages as the bot user, rather than as the app (a very subtle distinction)
		LinkNames: true, // Parse @names and #names in the welcome message but still allow other fancy formatting.
	}
	if err := h.client.CallMethod("chat.postMessage", message, nil); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	return nil
}

func (h *handler) getWelcome() (string, error) {
	message, err := ioutil.ReadFile(h.messagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %v", h.messagePath, err)
	}
	return string(message), nil
}
