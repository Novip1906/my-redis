FROM golang:1.25.1-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -v -o ./app ./cmd/my-redis/main.go

CMD ["./app"]