# Speed-Test

## TCP
Testing network speed between server and client via TCP protocol. TCP (Transmission Control Protocol) is a reliable, connection-oriented protocol that ensures data is delivered accurately and in order, making it ideal for applications where data integrity is critical. By measuring the speed of data transfer between two endpoints, this test provides valuable insights into network performance, including bandwidth, latency, and potential bottlenecks. Whether you're optimizing a local network or diagnosing internet connectivity issues, the TCP speed test is a fundamental tool for understanding and improving your network's efficiency.


## QUIC
Testing network speed between server and client via QUIC protocol. QUIC (Quick UDP Internet Connections) is a modern, high-performance transport protocol designed to reduce latency and improve connection reliability, especially in unstable network conditions. Built on UDP, QUIC integrates encryption by default and supports multiplexed streams, making it a powerful choice for real-time applications like video streaming and online gaming. This speed test helps evaluate how QUIC performs under various network conditions, providing insights into its efficiency and speed compared to traditional protocols.



## HOW TO RUN
### Run local
1. Add config
```
cp .env.example .env
```

2. Run server
```
go run cmd/tcp/main.go -t server
```

3. Run client
```
go run cmd/tcp/main.go -t client
```

### Run local via docker
1. Add config
```
cp .env.example .env
```

2. Run docker compose
```
compose_up_local
```


### Run from local client to public servdrd
1. Add config
```
cp .env.example .env
```

2. Run server on host with public ip
```
make compose_up_server
```

3. Run clien in local network
```
make compose_up_client
```