BINARY := timesheet
PKG := ./cmd/timesheet

.PHONY: build run vet fmt clean

build:
	go build -o bin/$(BINARY) $(PKG)

# Contoh: make run ARGS="-owner myorg -month 2026-05 -format md"
run:
	go run $(PKG) $(ARGS)

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin
