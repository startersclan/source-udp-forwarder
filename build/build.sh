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

if [ -z "${OUTBIN:-}" ]; then
    echo "OUTBIN must be set"
    exit 1
fi
if [ -z "${OS:-}" ]; then
    echo "OS must be set"
    exit 1
fi
if [ -z "${ARCH:-}" ]; then
    echo "ARCH must be set"
    exit 1
fi
if [ -z "${VERSION:-}" ]; then
    echo "VERSION must be set"
    exit 1
fi

export CGO_ENABLED=0
export GOARCH="${ARCH}"
export GOOS="${OS}"
export GO111MODULE=on
if [ -d vendor ]; then
    export GOFLAGS="-mod=vendor"
fi

# Install git if specified
if [ -n "$INSTALL_GIT" ]; then
    apk add --no-cache git
fi

go build -o "$OUTBIN" -ldflags "-s -w -extldflags \"-static\" -X $(go list -m)/cmd.VERSION=$VERSION -X $(go list -m)/cmd.COMMIT_SHA1=$COMMIT_SHA1 -X $(go list -m)/cmd.BUILD_DATE=$BUILD_DATE"
