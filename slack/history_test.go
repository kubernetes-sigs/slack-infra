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

import "testing"

func TestParseSlackTS(t *testing.T) {
	cases := map[string]int64{
		"1512085950.000216": 1512085950,
		"1512085950":        1512085950,
		"":                  0,
		"garbage":           0,
	}
	for in, want := range cases {
		if got := parseSlackTS(in); got != want {
			t.Errorf("parseSlackTS(%q) = %d, want %d", in, got, want)
		}
	}
}
