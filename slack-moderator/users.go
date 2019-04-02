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
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"sigs.k8s.io/slack-infra/slack"
)

func (h *handler) getDisplayName(id string) (string, error) {
	user, err := h.getUserInfo(id)
	if err != nil {
		return "", err
	}
	return user.Name, nil
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

func (h *handler) userHasModerationPowers(id string) (bool, error) {
	user, err := h.getUserInfo(id)
	if err != nil {
		log.Printf("Failed to look up moderation powers: %v\n", err)
		return false, err
	}
	return user.IsAdmin || user.IsOwner || user.IsPrimaryOwner, nil
}

func (h *handler) deactivateUser(interaction slackInteraction, targetUser string) error {
	result := struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}{}

	resp, err := http.Post("https://slack.com/api/users.admin.setInactive", "application/x-www-form-urlencoded", bytes.NewBufferString("token="+h.adminToken+"&user="+targetUser))
	if err != nil {
		return fmt.Errorf("couldn't deactivate user %s: %v", targetUser, err)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode json: %v", err)
	}
	if !result.Ok {
		return fmt.Errorf("couldn't update membership: %s", result.Error)
	}

	return nil
}
