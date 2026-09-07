.PHONY: build test vet format smoke check run

build:
	go build -o bin/portfolio-ssh ./cmd/portfolio-ssh

test:
	go test -race ./...

vet:
	go vet ./...

format:
	@test -z "$$(gofmt -l cmd internal)" || { gofmt -l cmd internal; exit 1; }

smoke: build
	./bin/portfolio-ssh -help

check: format test vet smoke

run:
	go run ./cmd/portfolio-ssh
