FROM golang:1.21 as build
WORKDIR /go/src/admiral
COPY . .
RUN make

FROM scratch as runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/src/admiral/out/ /
ENTRYPOINT ["/admiral"]
