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

// Minimal Google Cloud Storage access, implemented directly against the GCS JSON
// API over net/http rather than with cloud.google.com/go/storage.
//
// This is a deliberate choice: slack-infra keeps a very small dependency set
// (see go.mod), and the official Storage SDK pulls in a large transitive tree
// (gRPC, the google.golang.org/api stack, etc.) that would dwarf the rest of the
// module. All we need is get/put of a single small object, which is a couple of
// authenticated HTTP calls. If this ever grows to need retries, resumable
// uploads, listing, or generation preconditions, prefer switching to the SDK
// over reimplementing those here.
//
// Authentication uses the token from the GCE/GKE metadata server, which is how
// Workload Identity presents the CronJob's service account. This only works when
// running on GCP; locally, use a filesystem path for --store instead of a gs://
// URL.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const metadataTokenURL = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"

var gcsHTTP = &http.Client{Timeout: 60 * time.Second}

// gcsToken fetches an OAuth access token from the metadata server.
func gcsToken() (string, error) {
	req, err := http.NewRequest(http.MethodGet, metadataTokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := gcsHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("querying metadata server (not running on GCP?): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server returned %s", resp.Status)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("metadata server returned an empty access token")
	}
	return tok.AccessToken, nil
}

// gcsGet downloads an object. found is false (nil error) when it does not exist.
func gcsGet(bucket, object string) (data []byte, found bool, err error) {
	token, err := gcsToken()
	if err != nil {
		return nil, false, err
	}
	u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s?alt=media",
		url.PathEscape(bucket), url.PathEscape(object))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := gcsHTTP.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("GCS get gs://%s/%s failed: %s: %s", bucket, object, resp.Status, body)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// gcsPut uploads (and overwrites) an object.
func gcsPut(bucket, object string, data []byte) error {
	token, err := gcsToken()
	if err != nil {
		return err
	}
	u := fmt.Sprintf("https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s",
		url.PathEscape(bucket), url.QueryEscape(object))
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := gcsHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GCS put gs://%s/%s failed: %s: %s", bucket, object, resp.Status, body)
	}
	return nil
}
