tidy:
	go mod tidy -v

fmt:
	go fmt ./...

test:
	go test ./... -covermode=atomic -coverprofile=coverage.out

build:
	CGO_ENABLED=0 GOOS=linux go build -o out/admiral ./cmd && chmod +x out/admiral