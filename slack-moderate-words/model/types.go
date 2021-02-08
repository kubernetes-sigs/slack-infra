/*
Copyright 2021 The Kubernetes Authors.

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

package model

type Challenge struct {
	Challenge string `json:"challenge"`
}

type SlackEvent struct {
	Token       string   `json:"token"`
	TeamID      string   `json:"team_id"`
	APIAppID    string   `json:"api_app_id"`
	Event       Event    `json:"event"`
	Type        string   `json:"type"`
	AuthedTeams []string `json:"authed_teams"`
	EventID     string   `json:"event_id"`
	EventTime   int      `json:"event_time"`
	Challenge   string   `json:"challenge,omitempty"`
}

type Event struct {
	Type        string `json:"type"`
	Channel     string `json:"channel"`
	User        string `json:"user"`
	Text        string `json:"text"`
	TS          string `json:"ts"`
	EventTS     string `json:"event_ts"`
	ThreadTS    string `json:"thread_ts"`
	ChannelType string `json:"channel_type"`
	BotID       string `json:"bot_id"`
}

type FilterConfig []struct {
	Triggers []string `yaml:"triggers"`
	Action   string   `yaml:"action"`
	Message  string   `yaml:"message"`
}
