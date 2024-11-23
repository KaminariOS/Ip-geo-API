FROM golang:1.23.1 AS builder

RUN mkdir -p /app
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine

WORKDIR /app

COPY --from=builder /app/app .

CMD ["/app/app"]
