FROM golang:1.24.7 AS builder

ARG GOOS=linux
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ./webcalsync ./main.go


FROM gcr.io/distroless/base-debian13
WORKDIR /app
COPY --from=builder /app/webcalsync /app/webcalsync
COPY --from=builder /usr/share/zoneinfo/Europe/Berlin /usr/share/zoneinfo/Europe/Berlin

CMD ["/app/webcalsync"]
