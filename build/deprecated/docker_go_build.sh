#!/bin/sh
workspaceFolder=$( git rev-parse --show-toplevel )
TAG=$( git describe --abbrev=0 --tags )
COMMIT_SHA=$( git rev-parse HEAD )
BUILD_DATE=$( date '+%Y-%m-%dT%H:%M:%S%z' )
docker run --rm -it -u "$(id -u)" -v ~/.cache:/.cache -v ~/go:/go -v ${workspaceFolder}/app:/go/src/app -v ${workspaceFolder}/bin:/go/bin -w /go/src/app golang:1.12 /bin/sh -c "$( cat - <<EOF
GOOS=linux CGO_ENABLED=0 go build -o /go/bin/source_builder_exporter -v -ldflags "-s -w -extldflags \"-static\" -X main.VERSION=$TAG -X main.COMMIT_SHA1=$COMMIT_SHA -X main.BUILD_DATE=$BUILD_DATE" && ls -al /go/bin/source_builder_exporterEOF
)"
