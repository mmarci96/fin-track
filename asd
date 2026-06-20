❯ docker build -t fin-track-backend .
[+] Building 16.1s (13/14)                                                                       docker:default
 => [internal] load build definition from Dockerfile                                                       0.0s
 => => transferring dockerfile: 273B                                                                       0.0s
 => [internal] load metadata for docker.io/library/alpine:latest                                           0.4s
 => [internal] load metadata for docker.io/library/golang:1.26.4                                           0.4s
 => [internal] load .dockerignore                                                                          0.0s
 => => transferring context: 2B                                                                            0.0s
 => CACHED [stage-1 1/2] FROM docker.io/library/alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0b  0.0s
 => => resolve docker.io/library/alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a  0.0s
 => [internal] load build context                                                                          0.0s
 => => transferring context: 6.67kB                                                                        0.0s
 => [builder 1/7] FROM docker.io/library/golang:1.26.4@sha256:792443b89f65105abba56b9bd5e97f680a80074ac62  0.0s
 => => resolve docker.io/library/golang:1.26.4@sha256:792443b89f65105abba56b9bd5e97f680a80074ac62fc844a58  0.0s
 => CACHED [builder 2/7] WORKDIR /build                                                                    0.0s
 => CACHED [builder 3/7] COPY go.mod go.sum .                                                              0.0s
 => CACHED [builder 4/7] RUN go mod download                                                               0.0s
 => [builder 5/7] COPY . .                                                                                 0.1s
 => [builder 6/7] RUN ls -la                                                                               0.2s
 => ERROR [builder 7/7] RUN go build -o app ./cmd/fin-track/main.go                                       15.2s
------
 > [builder 7/7] RUN go build -o app ./cmd/fin-track/main.go:
11.42 # github.com/otiai10/gosseract/v2
11.42 tessbridge.cpp:5:10: fatal error: leptonica/allheaders.h: No such file or directory
11.42     5 | #include <leptonica/allheaders.h>
11.42       |          ^~~~~~~~~~~~~~~~~~~~~~~~
11.42 compilation terminated.
------
Dockerfile:12
--------------------
  10 |     RUN ls -la
  11 |     
  12 | >>> RUN go build -o app ./cmd/fin-track/main.go
  13 |     
  14 |     FROM alpine:latest
--------------------
ERROR: failed to build: failed to solve: process "/bin/sh -c go build -o app ./cmd/fin-track/main.go" did not co
mplete successfully: exit code: 1
❯ go mod tidy

