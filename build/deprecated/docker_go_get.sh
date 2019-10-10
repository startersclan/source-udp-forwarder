#!/bin/sh
workspaceFolder=$( git rev-parse --show-toplevel )
docker run --rm -it -u "$(id -u)" -v ~/go:/go -v ${workspaceFolder}/app:/go/src/app -w /go/src/app golang:1.12 /bin/sh -c 'go get -d -v ./...'
