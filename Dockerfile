# Step-1
FROM golang:1.23 AS builder

WORKDIR /app

COPY . .

RUN go build -o speedtest-tcp cmd/tcp/main.go

# Step-2
FROM debian:stable-slim

RUN apt update && apt install -y iproute2 net-tools netcat-openbsd vim tcpdump iptables procps iputils-ping nload pv  \
    curl iperf3

COPY --from=builder /app/speedtest-tcp /app/speedtest-tcp

WORKDIR /app

CMD ["./speedtest-tcp"]