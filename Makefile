default: mo
	go run .

mo:
	msgfmt -c -v po/default.pot -o mo/en_GB.utf8/LC_MESSAGES/default.mo

build:
	go build -o darkstation main.go

codestyle:
	go fmt ./...
	golangci-lint run

go-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.64.5

test:
	go test ./...

clean:
	rm -f darkstation
	rm -rf dist/

.PHONY: mo default build codestyle go-tools test clean
