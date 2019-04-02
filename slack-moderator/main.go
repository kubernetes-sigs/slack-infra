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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"sigs.k8s.io/slack-infra/slack"
)

type options struct {
	configPath string
}

func parseFlags() options {
	o := options{}
	flag.StringVar(&o.configPath, "config-path", "config.json", "Path to a file containing the slack config")
	flag.Parse()
	return o
}

func runServer(h *handler) {
	http.Handle("/webhook", h)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func loadAdminToken(path string) (string, error) {
	extraConf := struct {
		AdminToken string `json:"adminToken"`
	}{}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("couldn't open file: %v", err)
	}
	if err := json.Unmarshal(content, &extraConf); err != nil {
		return "", fmt.Errorf("couldn't parse config: %v", err)
	}
	return extraConf.AdminToken, nil
}

func main() {
	o := parseFlags()
	c, err := slack.LoadConfig(o.configPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", o.configPath, err)
	}
	adminToken, err := loadAdminToken(o.configPath)
	if err != nil {
		log.Fatalf("Failed to load admin token from %s: %v", o.configPath, err)
	}
	s := slack.New(c)

	h := &handler{client: s, adminToken: adminToken}
	runServer(h)
}
