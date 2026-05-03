
.PHONY: build test test-race cover lint fmt vet tidy mocks clean

build: 
	go build ./...

test: 
	go test ./...

test-race: 
	go test -race ./...

cover: 
	go test -coverprofile=coverage.out $$(go list ./... | grep -v /examples)
	go tool cover -html=coverage.out -o coverage.html

lint: 
	go tool golangci-lint run

fmt: 
	gofmt -w -s .

vet: 
	go vet ./...

tidy: 
	go mod tidy

mocks: 
	go tool mockery

clean: 
	rm -f coverage.out coverage.html
	rm -rf bin/
