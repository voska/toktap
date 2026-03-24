FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o toktap ./cmd/toktap

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/toktap /usr/local/bin/toktap

EXPOSE 8080

HEALTHCHECK --interval=2s --timeout=1s --start-period=1s --retries=2 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["toktap"]
