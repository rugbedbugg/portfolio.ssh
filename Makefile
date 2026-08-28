.PHONY: build test vet check run

build:
	go build -o bin/portfolio-ssh ./cmd/portfolio-ssh

test:
	go test ./...

vet:
	go vet ./...

check: test vet build

run:
	go run ./cmd/portfolio-ssh
