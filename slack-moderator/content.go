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
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/slack-infra/slack"
)

func (h *handler) removeUserContent(interaction slackInteraction, duration time.Duration, targetUser string) (removedFiles, remainingFiles, removedMessages, remainingMessages int, err error) {
	if duration > 48*time.Hour {
		return 0, 0, 0, 0, fmt.Errorf("unacceptably long content removal duration: %s", duration)
	}
	start := time.Now().Add(-duration)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		var err error
		removedFiles, remainingFiles, err = h.removeFilesFromUser(targetUser, start)
		if err != nil {
			log.Printf("Couldn't remove files: %v", err)
		}
		wg.Done()
	}()
	go func() {
		var err error
		removedMessages, remainingMessages, err = h.removeMessagesFromUser(targetUser, start)
		if err != nil {
			log.Printf("Couldn't remove messages: %v", err)
		}
		wg.Done()
	}()
	wg.Wait()

	err = nil
	return
}

func (h *handler) removeFilesFromUser(targetUser string, since time.Time) (removed, remaining int, err error) {
	page := 1
	var files []string
	for {
		f, hasMore, err := h.searchForFiles(targetUser, since, page)
		if err != nil {
			if len(files) == 0 {
				return 0, 0, err
			}
			log.Printf("Failed to fetch more files (already got %d): %v\n", len(files), err)
			break
		}
		files = append(files, f...)
		if !hasMore {
			break
		}
		page += 1
	}
	log.Printf("Got %d files to remove...\n", len(files))
	for _, v := range files {
		if err := h.removeFile(v); err != nil {
			log.Printf("Failed to remove file %s: %v\n", v, err)
			remaining += 1
		} else {
			removed += 1
		}
	}
	return removed, remaining, nil
}

func (h *handler) removeFile(id string) error {
	log.Printf("Removing file %s\n", id)

	for {
		err := h.client.CallMethod("files.delete", map[string]string{"file": id}, nil)
		if err == nil {
			return nil
		}
		if timeout, ok := err.(slack.ErrRateLimit); ok {
			log.Printf("Slack is rate limiting us, trying again in %s...\n", timeout.Wait)
			time.Sleep(timeout.Wait)
			continue
		}
		return err
	}
}

func (h *handler) searchForFiles(targetUser string, since time.Time, page int) ([]string, bool, error) {
	args := map[string]string{
		"query":    fmt.Sprintf("from:<@%s> after:%s", targetUser, dateBefore(since)),
		"count":    "100",
		"sort":     "timestamp",
		"sort_dir": "desc",
		"page":     strconv.Itoa(page),
	}
	log.Printf("Searching files for %q", args["query"])

	result := struct {
		Files struct {
			Matches []struct {
				ID      string `json:"id"`
				Created int64  `json:"created"`
				User    string `json:"user"`
			} `json:"matches"`
			Pagination struct {
				PageCount int `json:"page_count"`
			} `json:"pagination"`
		} `json:"files"`
	}{}

	if err := h.client.CallOldMethod("search.files", args, &result); err != nil {
		return nil, false, fmt.Errorf("failed to find files: %v", err)
	}

	files := make([]string, 0, len(result.Files.Matches))
	for _, v := range result.Files.Matches {
		if v.User != targetUser {
			log.Printf("Got unexpected file %s from user %s instead of target user %s", v.ID, v.User, targetUser)
			continue
		}
		if time.Unix(v.Created, 0).Before(since) {
			log.Printf("Got unexpected file %s created at %s, which is before %s", v.ID, time.Unix(v.Created, 0), since)
			break
		}
		files = append(files, v.ID)
	}
	return files, result.Files.Pagination.PageCount > page, nil
}

type messageID struct {
	ts      string
	channel string
}

func (h *handler) removeMessagesFromUser(targetUser string, since time.Time) (removed, remaining int, err error) {
	page := 1
	var messages []messageID
	for {
		m, hasMore, err := h.searchForMessages(targetUser, since, page)
		if err != nil {
			if len(messages) == 0 {
				return 0, 0, err
			}
			log.Printf("Failed to fetch more messages (already got %d): %v\n", len(messages), err)
			break
		}
		messages = append(messages, m...)
		if !hasMore {
			break
		}
		page += 1
	}
	log.Printf("Got %d messages to remove...\n", len(messages))
	for _, v := range messages {
		if err := h.removeMessage(v); err != nil {
			log.Printf("Failed to remove message %s: %v\n", v, err)
			remaining += 1
		} else {
			removed += 1
		}
	}
	return removed, remaining, nil
}

func (h *handler) removeMessage(message messageID) error {
	req := map[string]interface{}{
		"channel": message.channel,
		"ts":      message.ts,
		"as_user": true,
	}

	log.Printf("Removing message %s\n", message)

	for {
		err := h.client.CallMethod("chat.delete", req, nil)
		if err == nil {
			return nil
		}
		switch e := err.(type) {
		case slack.ErrRateLimit:
			log.Printf("Slack is rate limiting us, trying again in %s...\n", e.Wait)
			time.Sleep(e.Wait)
		case slack.ErrSlack:
			if e.Type == "message_not_found" {
				log.Printf("Message to delete not found, probably already deleted.\n")
				return nil
			}
			return err
		default:
			return err
		}
	}
}

// Because slack search can only search for messages *after* a specific date
// Then subtract another day because the timezone behaviour is wildly unclear.
func dateBefore(when time.Time) string {
	return when.Add(-2 * 24 * time.Hour).Format("2006-01-02")
}

func (h *handler) searchForMessages(targetUser string, since time.Time, page int) ([]messageID, bool, error) {
	args := map[string]string{
		"query":    fmt.Sprintf("from:<@%s> after:%s", targetUser, dateBefore(since)),
		"count":    "100",
		"sort":     "timestamp",
		"sort_dir": "desc",
		"page":     strconv.Itoa(page),
	}
	log.Printf("Searching messages for %q", args["query"])

	result := struct {
		Messages struct {
			Matches []struct {
				Channel struct {
					ID string `json:"id"`
				} `json:"channel"`
				TS   string `json:"ts"`
				User string `json:"user"`
			} `json:"matches"`
			Pagination struct {
				PageCount int `json:"page_count"`
			} `json:"pagination"`
		} `json:"messages"`
	}{}

	if err := h.client.CallOldMethod("search.messages", args, &result); err != nil {
		return nil, false, fmt.Errorf("failed to find messages: %v", err)
	}

	messages := make([]messageID, 0, len(result.Messages.Matches))
	for _, v := range result.Messages.Matches {
		if v.User != targetUser {
			log.Printf("Unexpected message %s/%s from user %s, not target user %s\n", v.Channel, v.TS, v.User, targetUser)
			continue
		}
		ts := strings.SplitN(v.TS, ".", 2)[0]
		t, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			log.Printf("Failed to parse timestamp %s: %v\n", ts, err)
			continue
		}
		if time.Unix(t, 0).Before(since) {
			log.Printf("Got message %s/%s posted %s, which is before %s, assuming we're done.", v.Channel, v.TS, time.Unix(t, 0), since)
			break
		}
		messages = append(messages, messageID{
			ts:      v.TS,
			channel: v.Channel.ID,
		})
	}
	return messages, result.Messages.Pagination.PageCount > page, nil
}
