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
	"time"
)

// SetUserInactive deactivates a user via the admin users.admin.setInactive
// endpoint. This is an undocumented legacy endpoint (the same one slack-moderator
// uses); the access token must belong to an admin or owner.
func (c *Client) SetUserInactive(userID string) error {
	args := map[string]string{"user": userID}
	for {
		if err := c.CallOldMethod("users.admin.setInactive", args, nil); err != nil {
			switch e := err.(type) {
			case ErrRateLimit:
				time.Sleep(e.Wait)
				continue
			default:
				return err
			}
		}
		return nil
	}
}
