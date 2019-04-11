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
	"flag"
	"log"
	"os"
	"path"

	"sigs.k8s.io/slack-infra/slack"
	"sigs.k8s.io/slack-infra/tempelis/config"
	"sigs.k8s.io/slack-infra/tempelis/reconciler"
)

type options struct {
	dryRun       bool
	config       string
	restrictions string
	authConfig   string
}

func parseOptions() options {
	var o options
	flag.BoolVar(&o.dryRun, "dry-run", true, "does nothing if true (which is the default)")
	flag.StringVar(&o.config, "config", "", "path to a configuration file, or directory of files")
	flag.StringVar(&o.restrictions, "restrictions", "", "path to a configuration file containing restrictions")
	flag.StringVar(&o.authConfig, "auth", "", "path to slack auth")
	flag.Parse()
	return o
}

func main() {
	o := parseOptions()

	sc, err := slack.LoadConfig(o.authConfig)
	if err != nil {
		log.Fatalf("Failed to load slack auth config: %v.\n", err)
	}

	stat, err := os.Stat(o.config)
	if err != nil {
		log.Fatalf("Failed to stat %s: %v\n", o.config, err)
	}
	p := config.NewParser()

	if o.restrictions != "" {
		if err := p.ParseFile(o.restrictions, path.Dir(o.restrictions)); err != nil {
			log.Fatalf("Failed to parse restrictions file: %v.\n", err)
		}
	}

	if stat.IsDir() {
		err = p.ParseDir(o.config)
	} else {
		err = p.ParseFile(o.config, path.Dir(o.config))
	}
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	r := reconciler.New(slack.New(sc), p.Config)
	if err := r.Reconcile(o.dryRun); err != nil {
		log.Fatalf("Reconciliation failed: %v\n", err)
	}
}
