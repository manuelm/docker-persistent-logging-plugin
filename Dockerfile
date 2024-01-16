FROM golang:1.21-alpine3.19 as builder

ENV GO111MODULE="on"

RUN mkdir /build
WORKDIR /build

# Copy golang dependency manifests
COPY src/go.mod .
COPY src/go.sum .

# Cache the dependency downloads
RUN go mod download

# Add source code and build
COPY ./src .
RUN go build --ldflags '-w -extldflags "-static"' -o persistent-logging-plugin

# Copy compiled plugin over into to slim image to reduced the size
FROM alpine:3.19
COPY --from=builder /build/persistent-logging-plugin /usr/bin/persistent-logging-plugin
