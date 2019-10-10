# Build stage
FROM golang:1.12-alpine AS builder
WORKDIR /src
COPY . .
RUN apk add --no-cache git
RUN go get -d -v ./...

ARG OS=linux
ARG ARCH=amd64
ARG BIN
ARG VERSION
ARG COMMIT_SHA1
ARG BUILD_DATE
RUN GOOS="$OS" GOARCH="$ARCH" CGO_ENABLED=0 go build -o "/go/bin/$BIN" -v -ldflags "-s -w -extldflags \"-static\" -X main.VERSION=$VERSION -X main.COMMIT_SHA1=$COMMIT_SHA1 -X main.BUILD_DATE=$BUILD_DATE"

# Final stage
FROM alpine:3.8 AS final

ARG BIN
ARG VERSION
ARG COMMIT_SHA1
ARG BUILD_DATE
LABEL NAME=$BIN
LABEL VERSION=$VERSION
LABEL COMMIT_SHA1=$COMMIT_SHA1
LABEL BUILD_DATE=$BUILD_DATE

RUN apk --no-cache add ca-certificates
ARG BIN
COPY --from=builder /go/bin/$BIN /$BIN
ENV BIN=$BIN
ENTRYPOINT /$BIN
EXPOSE 26999

