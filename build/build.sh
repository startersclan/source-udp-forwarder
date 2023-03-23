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

export CGO_ENABLED=0
export GOARCH="${ARCH}"
export GOOS="${OS}"
export GOFLAGS="$GOFLAGS"
export GO111MODULE=on

go build -o "$OUTBIN" -ldflags "-s -w -extldflags \"-static\" -X $(go list -m)/cmd.VERSION=$VERSION -X $(go list -m)/cmd.COMMIT_SHA1=$COMMIT_SHA1 -X $(go list -m)/cmd.BUILD_DATE=$BUILD_DATE"
