/*
Copyright The Kubernetes Authors.

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

package slack

import (
	"fmt"
	"time"
)

// GetUsers returns every user in the workspace via users.list, following cursor
// pagination. The returned users carry the flags (IsAdmin, IsOwner, IsBot,
// IsRestricted, Deleted, ...) needed to protect accounts from being pruned.
func (c *Client) GetUsers() ([]User, error) {
	var users []User
	cursor := ""
	for {
		args := map[string]string{
			"limit": "1000",
		}
		if cursor != "" {
			args["cursor"] = cursor
		}

		ret := struct {
			Members  []User `json:"members"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}{}

		for {
			if err := c.CallOldMethod("users.list", args, &ret); err != nil {
				switch e := err.(type) {
				case ErrRateLimit:
					time.Sleep(e.Wait)
					continue
				default:
					return nil, fmt.Errorf("failed to list users: %v", err)
				}
			}
			break
		}

		users = append(users, ret.Members...)
		if ret.Metadata.NextCursor == "" {
			break
		}
		cursor = ret.Metadata.NextCursor
	}
	return users, nil
}
