# syntax=docker/dockerfile:1

ARG ARCH
FROM golang:1.20 as builder

WORKDIR /tmp/build

ENV CGO_ENABLED=0

COPY . .
RUN go build -o dist/serviceFabricDiscoveryService ./cmd

FROM gcr.io/distroless/static

COPY --from=builder /tmp/build/dist/serviceFabricDiscoveryService /

ENTRYPOINT ["/serviceFabricDiscoveryService"]
