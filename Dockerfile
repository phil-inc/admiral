FROM golang:1.17 as build
WORKDIR /go/src/admiral
COPY . .
RUN make build

FROM ubuntu:22.04 as runtime
RUN apt update && apt install ca-certificates && apt upgrade
COPY --from=build /go/src/admiral/out/ /bin/
ENTRYPOINT ["/bin/admiral"]