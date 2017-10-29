package restserver

import "github.com/prometheus/client_golang/prometheus"

var metricLabelList = []string{"user", "repo", "type"}

var metricBlobWriteTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_write_total",
		Help: "Total number of blobs written",
	},
	metricLabelList,
)

var metricBlobWriteBytesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_write_bytes_total",
		Help: "Total number of bytes written to blobs",
	},
	metricLabelList,
)

var metricBlobReadTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_read_total",
		Help: "Total number of blobs read",
	},
	metricLabelList,
)

var metricBlobReadBytesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_read_bytes_total",
		Help: "Total number of bytes read from blobs",
	},
	metricLabelList,
)

var metricBlobDeleteTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_delete_total",
		Help: "Total number of blobs deleted",
	},
	metricLabelList,
)

var metricBlobDeleteBytesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rest_server_blob_delete_bytes_total",
		Help: "Total number of bytes of blobs deleted",
	},
	metricLabelList,
)

func init() {
	// These are always initialized, but only updated if Config.Prometheus is set
	prometheus.MustRegister(metricBlobWriteTotal)
	prometheus.MustRegister(metricBlobWriteBytesTotal)
	prometheus.MustRegister(metricBlobReadTotal)
	prometheus.MustRegister(metricBlobReadBytesTotal)
	prometheus.MustRegister(metricBlobDeleteTotal)
	prometheus.MustRegister(metricBlobDeleteBytesTotal)
}
