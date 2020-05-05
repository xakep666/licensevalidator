ARG GO_VERSION=1.14
ARG ALPINE_VERSION=3.9

FROM golang:${GO_VERSION}-alpine AS builder
ARG GOPROXY="https://proxy.golang.org"
ENV GOPROXY="$GOPROXY"
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /go/licensevalidator
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -v -ldflags="-w -s" -o /bin/licensevalidator ./cmd/licensevalidator

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/licensevalidator /bin/licensevalidator
RUN licensevalidator sample-config > /etc/licensevalidator.toml

CMD ["licensevalidator", "--config=/etc/licensevalidator.toml"]
