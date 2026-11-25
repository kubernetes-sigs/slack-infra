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

package main

import (
	"encoding/json"
	"errors"
	"os"
)

// checkpointVersion is bumped if the on-disk format changes incompatibly.
const checkpointVersion = 1

// walkCheckpoint is the serialized state of an access-log walk. It captures both
// the accumulated results (LastActive, counters) and the exact cursor position
// (Before, Page, WindowMin) so a walk that is killed part way through can resume
// instead of starting over. CutoffTS and PageSize are recorded so a resume with
// mismatched settings can be rejected rather than silently corrupting results.
type walkCheckpoint struct {
	Version   int   `json:"version"`
	CutoffTS  int64 `json:"cutoffTS"`
	PageSize  int   `json:"pageSize"`
	Before    int64 `json:"before"`
	Page      int   `json:"page"`
	WindowMin int64 `json:"windowMin"`

	APICalls     int   `json:"apiCalls"`
	TotalEntries int   `json:"totalEntries"`
	OldestSeen   int64 `json:"oldestSeen"`

	ReachedCutoff bool `json:"reachedCutoff"`
	Exhausted     bool `json:"exhausted"`
	// Complete is true once the walk has fully finished (reached the cutoff or
	// exhausted the logs). A complete checkpoint is reported as-is on the next
	// run instead of walking again.
	Complete bool `json:"complete"`

	LastActive map[string]int64  `json:"lastActive"`
	Usernames  map[string]string `json:"usernames,omitempty"`
}

// loadCheckpoint reads a checkpoint from path. It returns (nil, nil) if the file
// does not exist, so callers can treat "no checkpoint" as "start fresh".
func loadCheckpoint(path string) (*walkCheckpoint, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cp := &walkCheckpoint{}
	if err := json.Unmarshal(data, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

// saveCheckpoint writes a checkpoint to path atomically (write to a temp file in
// the same directory, then rename) so a crash mid-write can't corrupt it.
func saveCheckpoint(path string, cp *walkCheckpoint) error {
	data, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
