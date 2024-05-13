FROM quay.io/projectquay/golang:1.20 as build

WORKDIR /go/src/app
COPY . .
ARG TARGETARCH
RUN make build TARGETARCH=$TARGETARCH

FROM scratch
WORKDIR /
COPY --from=build /go/src/app/abot .
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["./abot", "go"]
