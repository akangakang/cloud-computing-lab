FROM golang:1.16-buster

WORKDIR /go/src/gDoc_backend

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build

EXPOSE 8090

CMD ["./gDoc_backend"]