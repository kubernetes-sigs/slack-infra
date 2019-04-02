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

package slack

import (
	"fmt"
	"time"
)

type ErrRateLimit struct {
	Wait time.Duration
}

func (e ErrRateLimit) Error() string {
	return fmt.Sprintf("slack has rate limited us for the next %s", e.Wait)
}

type ErrSlack struct {
	Type     string
	Warnings []string
}

func (e ErrSlack) Error() string {
	return fmt.Sprintf("slack call failed: %s (%v)", e.Type, e.Warnings)
}
