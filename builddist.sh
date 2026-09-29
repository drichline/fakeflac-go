#!/bin/sh

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <version>" >&2
    exit 1
fi

mkdir -p dist
rm -r dist/*
cd dist

# macOS
mkdir -p mac
GOOS=darwin GOARCH=amd64 go build -o mac/fakeflac-go-$1-amd64 ..
GOOS=darwin GOARCH=arm64 go build -o mac/fakeflac-go-$1-arm64 ..
zip fakeflac-go_darwin.zip mac/fakeflac-go-$1-*
rm -r mac

# Windows
GOOS=windows GOARCH=amd64 go build -o fakeflac-go-$1.exe ..
zip fakeflac-go_windows_amd64.zip fakeflac-go-$1.exe
rm fakeflac-go-$1.exe

# Linux amd64
GOOS=linux GOARCH=amd64 go build -o fakeflac-go-$1 ..
zip fakeflac-go_linux_amd64.zip fakeflac-go-$1
rm fakeflac-go-$1

# Linux arm
mkdir -p arm
GOOS=linux GOARCH=arm GOARM=6 go build -o arm/fakeflac-go-$1-armv6 ..
GOOS=linux GOARCH=arm GOARM=7 go build -o arm/fakeflac-go-$1-armv7 ..
GOOS=linux GOARCH=arm64 go build -o arm/fakeflac-go-$1-arm64 ..
zip fakeflac-go_linux_arm.zip arm/*
rm -r arm