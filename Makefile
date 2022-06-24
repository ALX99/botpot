curr_dir = $(shell pwd)

build:
	go build -v -a -ldflags "-s -w" -o ./build/botpot ./cmd/botpot/main.go