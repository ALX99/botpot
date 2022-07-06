curr_dir = $(shell pwd)

build:
	go build -v -a -ldflags "-s -w" -o ./build/botpot ./cmd/botpot/main.go

build-release:
	CGO_ENABLED=0 go build -v -a -ldflags "-s -w" -o ./build/botpot ./cmd/botpot/main.go

image: build-release
	docker build . -t botpot:latest

run:
	go run ./cmd/botpot/main.go