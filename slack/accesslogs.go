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
	"strconv"
	"time"
)

// AccessLogin is a single entry from team.accessLogs. Each entry represents a
// distinct (user, IP, user-agent) combination rather than a single user, so a
// given user may appear in many entries.
//
// See https://docs.slack.dev/reference/methods/team.accessLogs
type AccessLogin struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Count     int    `json:"count"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	ISP       string `json:"isp"`
	Country   string `json:"country"`
	Region    string `json:"region"`
	// DateFirst and DateLast are Unix timestamps (seconds) for the first and
	// most recent access for this entry.
	DateFirst int64 `json:"date_first"`
	DateLast  int64 `json:"date_last"`
}

// AccessLogsPage is one page of results from team.accessLogs.
type AccessLogsPage struct {
	Logins []AccessLogin
	// Pages is the total number of pages available for the current query
	// window, as reported by Slack's paging metadata.
	Pages int
}

// GetAccessLogsPage fetches a single page of team.accessLogs.
//
// before, if non-zero, restricts results to entries at or before the given
// Unix timestamp (seconds), which is how you page backwards through history
// beyond the 100-page-per-query limit. count is the page size (max 1000) and
// page is the 1-indexed page number (max 100).
//
// This method requires an admin-scoped user token in Config.AccessToken and is
// only available on paid workspaces (free workspaces return a paid_only error).
// Rate limits (Tier 2) are handled by retrying after the requested delay.
func (c *Client) GetAccessLogsPage(before int64, count, page int) (AccessLogsPage, error) {
	args := map[string]string{
		"count": strconv.Itoa(count),
		"page":  strconv.Itoa(page),
	}
	if before > 0 {
		args["before"] = strconv.FormatInt(before, 10)
	}

	ret := struct {
		Logins []AccessLogin `json:"logins"`
		Paging struct {
			Count int `json:"count"`
			Total int `json:"total"`
			Page  int `json:"page"`
			Pages int `json:"pages"`
		} `json:"paging"`
	}{}

	for {
		if err := c.CallOldMethod("team.accessLogs", args, &ret); err != nil {
			switch e := err.(type) {
			case ErrRateLimit:
				time.Sleep(e.Wait)
				continue
			default:
				return AccessLogsPage{}, fmt.Errorf("failed to get access logs (before=%d page=%d): %v", before, page, err)
			}
		}
		break
	}

	return AccessLogsPage{Logins: ret.Logins, Pages: ret.Paging.Pages}, nil
}
