curr_dir = $(shell pwd)

.PHONY: clean image keys

build:
	CGO_ENABLED=1 go build -race -v -trimpath -ldflags "-s -w" -o ./build/botpot ./cmd/botpot/main.go

build-release:
	CGO_ENABLED=0 go build -v -a -trimpath "-s -w" -o ./build/botpot ./cmd/botpot/main.go

image:
	docker build . -t botpot:latest
	docker build . --file sshpot/Dockerfile -t sshpot:latest
	docker build db/ -t sshpotdb:latest

run:
	CGO_ENABLED=1 go run -race ./cmd/botpot/main.go

keys:
	ssh-keygen -t rsa -N "" -f ./keys/rsa.pem
	ssh-keygen -t ed25519 -N "" -f ./keys/ed25519.pem
	ssh-keygen -t ecdsa -b 256 -N "" -f ./keys/ecdsa256.pem
	ssh-keygen -t ecdsa -b 384 -N "" -f ./keys/ecdsa384.pem
	ssh-keygen -t ecdsa -b 521 -N "" -f ./keys/ecdsa521.pem

clean:
	rm -rf ./build
