# syntax=docker/dockerfile:1

FROM docker.io/golang:latest AS builder

WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux GOBIN=/app go install github.com/drichline/fakeflac-go@v0.1.1

FROM docker.io/alpine:latest

RUN apk add --no-cache ffmpeg

COPY --from=builder /app/fakeflac-go /bin/fakeflac-go

RUN mkdir /files

WORKDIR /files

ENTRYPOINT ["/bin/fakeflac-go", "-ffmpeg"]