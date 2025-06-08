FROM golang:1.24.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:3.22.0

WORKDIR /app

VOLUME ["/app/data"]

COPY --from=builder /app/server .

CMD ["/app/server"]
