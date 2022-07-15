curr_dir = $(shell pwd)

build:
	CGO_ENABLED=1 go build -race -v -a -ldflags "-s -w" -o ./build/botpot ./cmd/botpot/main.go

build-release:
	CGO_ENABLED=0 go build -v -a -ldflags "-s -w" -o ./build/botpot ./cmd/botpot/main.go

image: build
	docker build . -t botpot:latest
	docker build . --file sshpot/Dockerfile -t sshpot:latest

run:
	go run ./cmd/botpot/main.go

keys: ed25519.pem rsa.pem
	ssh-keygen -t rsa -N "" -f rsa.pem
	ssh-keygen -t ed25519 -N "" -f ed25519.pem
	ssh-keygen -t ecdsa -b 256 -N "" -f ecdsa256.pem
	ssh-keygen -t ecdsa -b 384 -N "" -f ecdsa384.pem
	ssh-keygen -t ecdsa -b 521 -N "" -f ecdsa521.pem


clean:
	rm -rf ./build