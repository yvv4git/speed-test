package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bytesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tcp_server_bytes_received_total",
		Help: "Total number of bytes received from clients.",
	})

	bytesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tcp_server_bytes_sent_total",
		Help: "Total number of bytes sent to clients.",
	})
)

func startMetricsWebServer(cfg ServerConfig) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(cfg.MetricsAddr, nil)
}
