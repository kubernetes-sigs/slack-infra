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

package reconciler

import (
	"fmt"
	"log"

	"sigs.k8s.io/slack-infra/slack"
	"sigs.k8s.io/slack-infra/tempelis/config"
)

type Reconciler struct {
	slack    *slack.Client
	config   config.Config
	channels channelState
}

func New(slack *slack.Client, config config.Config) *Reconciler {
	return &Reconciler{
		slack:    slack,
		config:   config,
		channels: channelState{},
	}
}

func (r *Reconciler) Reconcile(dryRun bool) error {
	if err := r.channels.init(r.slack); err != nil {
		return fmt.Errorf("failed to get initial channel state: %v", err)
	}
	var actions []Action
	var errors []error
	a, e := r.reconcileChannels()
	actions = append(actions, a...)
	errors = append(errors, e...)
	a, e = r.reconcileUsergroups()
	actions = append(actions, a...)
	errors = append(errors, e...)

	failed := false
	if len(errors) > 0 {
		log.Printf("This configuration cannot be applied against the current reality:")
		failed = true
	}

	for i, e := range errors {
		log.Printf("Error %d: %v.\n", i+1, e)
	}

	if !dryRun && failed {
		dryRun = true
		log.Printf("We will not execute anything due to errors, but this what we would've done:")
	}

	for i, a := range actions {
		log.Printf("Step %d: %s.\n", i+1, a.Describe())
		if !dryRun {
			if err := a.Perform(r); err != nil {
				log.Printf("Failed: %v.\n", err)
			}
		}
	}

	if failed {
		return fmt.Errorf("there were configuration errors")
	}
	return nil
}

type Action interface {
	Describe() string
	Perform(reconciler *Reconciler) error
}
