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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"

	"sigs.k8s.io/slack-infra/slack"
	"sigs.k8s.io/slack-infra/slack-moderator-words/model"
)

type options struct {
	configPath       string
	filterConfigPath string
}

func parseFlags() options {
	o := options{}
	flag.StringVar(&o.configPath, "config-path", "config.json", "Path to a file containing the slack config")
	flag.StringVar(&o.filterConfigPath, "filter-config-path", "filters.yaml", "Path to a file containing the filter config")
	flag.Parse()
	return o
}

func runServer(h *handler) error {
	http.Handle(os.Getenv("PATH_PREFIX")+"/webhook", h)
	http.HandleFunc("/healthz", handleHealthz)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8077"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	return http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func loadFilterConfig(path string) (model.FilterConfig, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open file: %v", err)
	}
	filterConfig := &model.FilterConfig{}
	if err := yaml.Unmarshal(content, filterConfig); err != nil {
		return nil, fmt.Errorf("couldn't parse filter config: %v", err)
	}

	return *filterConfig, nil
}

func main() {
	o := parseFlags()
	c, err := slack.LoadConfig(o.configPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", o.configPath, err)
	}

	filters, err := loadFilterConfig(o.filterConfigPath)
	if err != nil {
		log.Fatalf("Failed to load filter config from %s: %v", o.filterConfigPath, err)
	}

	s := slack.New(c)

	// List all public channels and try to join.
	// This is needed otherwise the bot cannot receive the events for the channels
	// and cannot moderate it
	channels, err := s.GetPublicChannels()
	if err != nil {
		log.Fatalf("Failed to list all public channels: %v", err)
	}

	for _, channel := range channels {
		if channel.IsArchived {
			log.Printf("Public Channel: %s/%s is archived, skipping...\n", channel.ID, channel.Name)
			continue
		}
		log.Printf("Public Channels: %s/%s\n", channel.ID, channel.Name)
		req := map[string]interface{}{
			"channel": channel.ID,
		}
		err = s.CallMethod("conversations.join", req, nil)
		if err != nil {
			log.Fatalf("Failed to join channel %s: %v", channel.Name, err)
		}
	}

	h := &handler{client: s, filters: filters}
	log.Fatal(runServer(h))
}
