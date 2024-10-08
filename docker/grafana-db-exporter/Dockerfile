FROM golang:1.23.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o exporter ./cmd/exporter

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/exporter .

CMD ["./exporter"]
