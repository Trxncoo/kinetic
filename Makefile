.PHONY: build test race bench bench-graph vet fmt lint tidy check

COUNT ?= 10
PKG ?= ./...

build:
	go build ./...

test:
	go test ./...

race:
	go test ./... -race

bench:
	go test $(PKG) -bench=. -benchmem -run=^$$ -cpu=1,4,8

bench-graph:
	go test $(PKG) -bench=. -benchmem -run=^$$ -cpu=1,4,8 -benchtime=100ms -count=$(COUNT) | python3 scripts/benchgraph.py

vet:
	go vet ./...

fmt:
	gofmt -l .

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

check: vet race lint
