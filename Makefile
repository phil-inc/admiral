tidy:
	go mod tidy -v

# go fmt
fmt:
	gofmt -s -w .

# go fmt list files affected
fmt_list:
	gofmt -s -l .


test:
	go test ./... -covermode=atomic -coverprofile=coverage.out

build:
	CGO_ENABLED=0 GOOS=linux go build -o out/admiral ./cmd && chmod +x out/admiral