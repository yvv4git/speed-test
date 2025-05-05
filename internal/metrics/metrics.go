package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bytesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bytes_received_total",
		Help: "Total number of bytes received from clients.",
	})

	bytesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bytes_sent_total",
		Help: "Total number of bytes sent to clients.",
	})
)

func StartMetricsWebServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}

func AddBytesReceived(n int) {
	bytesReceived.Add(float64(n))
}

func AddBytesSent(n int) {
	bytesSent.Add(float64(n))
}
