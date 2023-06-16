build: tidy fmt test
	CGO_ENABLED=0 GOOS=linux go build -o out/admiral ./cmd && chmod +x out/admiral

test:
	go test ./... -cover -v

fmt:
	gofmt -s -w .

tidy:
	go mod tidy -v
