#!/bin/sh

# Copyright 2016 The Kubernetes Authors.
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

set -eu

export CGO_ENABLED=1
export GOFLAGS="${GOFLAGS:-}"
export GO111MODULE=on


echo "Running tests:"
COVERAGE_FILE=${COVERAGE_FILE:-}
if [ -n "$COVERAGE_FILE" ]; then
    go test -v -race -coverprofile=$COVERAGE_FILE -covermode=atomic "$@"
else
    go test -v -race "$@"
fi
echo

PKG=${PKG:-$(go list "$@" | xargs echo)}

echo -n "Checking gofmt: "
ERRS=$(go fmt $PKG)
if [ -n "${ERRS}" ]; then
    echo "FAIL - the following files need to be gofmt'ed:"
    echo "${ERRS}"
    exit 1
fi
echo "PASS"
echo

echo -n "Checking go vet: "
ERRS=$(go vet $PKG)
if [ -n "${ERRS}" ]; then
    echo "FAIL"
    echo "${ERRS}"
    exit 1
fi
echo "PASS"
echo
