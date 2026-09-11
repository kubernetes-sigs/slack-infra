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
	"sigs.k8s.io/slack-infra/slack"
)

type usergroupState struct {
	byHandle map[string]*slack.Subteam
	byID     map[string]*slack.Subteam
}

func (u *usergroupState) init(s *slack.Client) error {
	u.byHandle = map[string]*slack.Subteam{}
	u.byID = map[string]*slack.Subteam{}

	result := struct {
		Usergroups []slack.Subteam `json:"usergroups"`
	}{}
	if err := s.CallOldMethod("usergroups.list", map[string]string{"include_users": "true", "include_disabled": "true"}, &result); err != nil {
		return fmt.Errorf("couldn't get usergroup list: %v", err)
	}

	for _, ug := range result.Usergroups {
		ug2 := ug
		u.byHandle[ug.Handle] = &ug2
		u.byID[ug.ID] = &ug2
	}
	return nil
}

func (u *usergroupState) create(handle string) error {
	if _, ok := u.byHandle[handle]; ok {
		return fmt.Errorf("can't create usergroup %s: name already used", handle)
	}
	g := slack.Subteam{
		Handle: handle,
	}
	u.byHandle[handle] = &g
	return nil
}
