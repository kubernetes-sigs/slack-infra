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
	"strings"
	"time"
)

// GetLastPostByUser sweeps a conversation's message history newer than `oldest`
// (Unix seconds) and returns each posting user's most recent message time in
// Unix seconds. Thread replies are not included (that would need a
// conversations.replies call per thread). Use it to tell that an account is
// still active even when the access logs show no recent session activity.
func (c *Client) GetLastPostByUser(channelID string, oldest int64) (map[string]int64, error) {
	lastPost := map[string]int64{}
	cursor := ""
	for {
		args := map[string]string{
			"channel": channelID,
			"limit":   "999",
			"oldest":  strconv.FormatInt(oldest, 10),
		}
		if cursor != "" {
			args["cursor"] = cursor
		}

		ret := struct {
			Messages []struct {
				User string `json:"user"`
				TS   string `json:"ts"`
			} `json:"messages"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}{}

		for {
			if err := c.CallOldMethod("conversations.history", args, &ret); err != nil {
				switch e := err.(type) {
				case ErrRateLimit:
					time.Sleep(e.Wait)
					continue
				default:
					return nil, fmt.Errorf("failed to read conversation history: %v", err)
				}
			}
			break
		}

		for _, m := range ret.Messages {
			if m.User == "" {
				continue // bot/system message with no author
			}
			if ts := parseSlackTS(m.TS); ts > lastPost[m.User] {
				lastPost[m.User] = ts
			}
		}

		if ret.Metadata.NextCursor == "" {
			break
		}
		cursor = ret.Metadata.NextCursor
	}
	return lastPost, nil
}

// parseSlackTS converts a Slack message timestamp ("1512085950.000216") to Unix
// seconds, returning 0 if it can't be parsed.
func parseSlackTS(ts string) int64 {
	sec, err := strconv.ParseInt(strings.SplitN(ts, ".", 2)[0], 10, 64)
	if err != nil {
		return 0
	}
	return sec
}
