package restserver

import "github.com/prometheus/client_golang/prometheus"


var metricBlobWriteTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_write_total",
		Help: "Total number of blobs written",
	},
	[]string{"repo", "type"},
)

var metricBlobWriteBytesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_write_bytes_total",
		Help: "Total number of bytes written to blobs",
	},
	[]string{"repo", "type"},
)

var metricBlobReadTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_read_total",
		Help: "Total number of blobs read",
	},
	[]string{"repo", "type"},
)

var metricBlobReadBytesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_read_bytes_total",
		Help: "Total number of bytes read from blobs",
	},
	[]string{"repo", "type"},
)

func init() {
	// These are always initialized, but only updated if Config.Prometheus is set
	prometheus.MustRegister(metricBlobWriteTotal)
	prometheus.MustRegister(metricBlobWriteBytesTotal)
	prometheus.MustRegister(metricBlobReadTotal)
	prometheus.MustRegister(metricBlobReadBytesTotal)
}
