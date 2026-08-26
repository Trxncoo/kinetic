.PHONY: build test race bench bench-graph vet fmt lint tidy check

COUNT ?= 10
# Only bench/bench-graph honor PKG — targeting one package's benchmarks
# (`make bench PKG=./pkg/kcache`) is common; targeting one package's
# tests/vet less so, so build/test/race/vet stay hardcoded to ./....
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
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then \
		echo "The following files are not gofmt'd:"; \
		echo "$$out"; \
		exit 1; \
	fi

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

check: fmt vet race lint
