# Copyright The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.DEFAULT_GOAL := help

.PHONY: help build test fmt vet verify-fmt verify

help: ## Show this help
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-12s %s\n", $$1, $$2}'

build: ## Check that all packages compile
	go build ./...

test: ## Run unit tests
	go test ./...

fmt: ## Format Go code with gofmt
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

verify-fmt: ## Check Go code formatting
	@out=$$(gofmt -s -l .); \
	if [ -n "$$out" ]; then \
		echo "These files need formatting, run 'make fmt':"; \
		echo "$$out"; \
		exit 1; \
	fi

verify: ## Run all verify checks (vet, gofmt) and report every failure
	@$(MAKE) --no-print-directory -k vet verify-fmt
