FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/thmnzr ./cmd/thmnzr

FROM alpine:3.22

RUN addgroup -S thmnzr && \
    adduser -S -G thmnzr thmnzr && \
    apk add --no-cache ca-certificates

COPY --from=builder /out/thmnzr /usr/local/bin/thmnzr

USER thmnzr
WORKDIR /work

CMD ["thmnzr", "--help"]
