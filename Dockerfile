FROM golang:1.24.7 AS builder

WORKDIR /app

COPY go.mod go.sum ./

COPY . .

RUN go build -o /sync

FROM alpine
COPY --from=builder /sync /sync

CMD ["/sync"]
