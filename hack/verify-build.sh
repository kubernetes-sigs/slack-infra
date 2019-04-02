#!/bin/bash

# Copyright 2019 The Kubernetes Authors.
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

# Tries to build all the executables to verify that they can in fact be built.

PACKAGES=(
  "slack-event-log"
  "slack-moderator"
  "slack-report-message"
  "slack-welcomer"
)

go-build() {
  target=$(mktemp)
  pushd "./$1" > /dev/null
  go build -o "${target}"
  ret=$?
  popd > /dev/null
  rm -f "${target}"
  return ${ret}
}

ret=0
for pkg in "${PACKAGES[@]}"; do
  go-build "${pkg}"
  result=$?
  if [[ ${result} -ne "0" ]]; then
    echo "Failed to build ${pkg}"
    ret=1
  fi
done

exit ${ret}
