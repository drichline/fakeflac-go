#!/bin/sh

mkdir -p dist
cd dist

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o fakeflac-go-$1 ..
zip fakeflac-go_darwin_amd64.zip fakeflac-go-$1
rm fakeflac-go-$1

# macOS Apple Silicon
# GOOS=darwin GOARCH=arm64 go build -o myprog-darwin-arm64 .
GOOS=darwin GOARCH=arm64 go build -o fakeflac-go-$1 ..
zip fakeflac-go_darwin_arm64.zip fakeflac-go-$1
rm fakeflac-go-$1

# Windows
# GOOS=windows GOARCH=amd64 go build -o myprog-windows-amd64.exe .
GOOS=windows GOARCH=amd64 go build -o fakeflac-go-$1.exe ..
zip fakeflac-go_windows_amd64.zip fakeflac-go-$1.exe
rm fakeflac-go-$1.exe

# Linux
# GOOS=linux GOARCH=amd64 go build -o myprog-linux-amd64 .
GOOS=linux GOARCH=amd64 go build -o fakeflac-go-$1 ..
zip fakeflac-go_linux_amd64.zip fakeflac-go-$1
rm fakeflac-go-$1