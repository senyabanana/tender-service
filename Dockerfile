FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main cmd/main.go

FROM alpine

WORKDIR /app

COPY --from=builder /app/main .
COPY app.env .
COPY migration ./migration

EXPOSE 8080

CMD ["/bin/sh", "-c", "source /app/app.env && /app/main"]