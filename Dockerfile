FROM golang:1.25 AS builder

WORKDIR /newApp

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o delayed_notifier ./cmd/main.go

EXPOSE 4051

CMD ["./delayed_notifier"]