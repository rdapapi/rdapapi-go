.PHONY: test lint fmt vet cover vuln canary

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

# Probes production through the public API. Needs RDAPAPI_API_KEY.
canary:
	go test -tags canary -count=1 -v -run TestCanary .
