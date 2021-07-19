##
## Build
##

FROM golang:1.16-buster AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY ./config ./config
COPY ./datanode ./datanode
COPY ./protocol ./protocol
COPY ./common_journal ./common_journal
COPY ./election ./election

RUN cd datanode && go build

WORKDIR /app/datanode
RUN chmod +x ./docker-entrypoint.sh

CMD ["./docker-entrypoint.sh"]