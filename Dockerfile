FROM  golang:1.15 as builder

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
RUN go build --ldflags '-extldflags "-static"' -o persistent-logging-plugin

# Copy compiled plugin over into to slim image to reduced the size
FROM debian:buster-slim
COPY --from=builder /build/persistent-logging-plugin /usr/bin/persistent-logging-plugin
