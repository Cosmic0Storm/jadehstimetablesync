FROM golang:1.24.7 AS builder

ARG GOOS=linux
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ./webcalsync ./main.go


FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/webcalsync /app/webcalsync

CMD ["/app/webcalsync"]
