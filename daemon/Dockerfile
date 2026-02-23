FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o myfeed-daemon .

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/myfeed-daemon /app/myfeed-daemon

RUN mkdir -p /data

ENV MYFEED_DATA_DIR=/data

EXPOSE 0

ENTRYPOINT ["/app/myfeed-daemon"]
CMD ["-data", "/data"]
