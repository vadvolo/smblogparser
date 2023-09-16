build:
	@go build -o bin/smblogparser

run: build
	@./bin/smblogparser

test:
	@go test -v ./...
