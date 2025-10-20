build:
	@go build -o bin/smblogparser

run: build
	@./bin/smblogparser -config config.yaml

test:
	@go test -v ./...

clean:
	@rm -rf bin/

install: build
	@cp bin/smblogparser /usr/local/bin/

.PHONY: build run test clean install
