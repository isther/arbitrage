FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY . .

RUN go build -o cmd bin/cmd/main.go

FROM ubuntu:18.04 as runner

COPY --from=builder /build/cmd /
COPY --from=builder /build/config.yaml /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y build-essential curl npm
RUN npm install n -g
RUN n lts

EXPOSE 8080

ENTRYPOINT ["/cmd"]
