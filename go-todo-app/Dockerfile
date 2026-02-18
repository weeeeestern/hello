FROM golang:1.22-bookworm

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /go-todo .

RUN apt-get update && apt-get install -y netcat-openbsd && rm -rf /var/lib/apt/lists/*

RUN chmod +x /app/wait-for.sh && cp /app/wait-for.sh /wait-for.sh

EXPOSE 8080

CMD ["/go-todo"]