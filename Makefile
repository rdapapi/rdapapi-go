.PHONY: test lint fmt vet cover vuln

test:
	go test -race -count=1 ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run

fmt:
	gofumpt -w .

vet:
	go vet ./...

vuln:
	govulncheck ./...
