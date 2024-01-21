package restserver

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/restic/rest-server/repo"
)

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

var metricRepoLastUpdateTimestamp = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "rest_server_repo_last_update_timestamp",
		Help: "Unix timestamp of repository's last write update",
	},
	[]string{"repo"},
)

// makeBlobMetricFunc creates a metrics callback function that increments the
// Prometheus metrics.
func makeBlobMetricFunc(username string, folderPath []string) repo.BlobMetricFunc {
	var f repo.BlobMetricFunc = func(objectType string, operation repo.BlobOperation, payload uint64) {
		repoPath := strings.Join(folderPath, "/")
		labels := prometheus.Labels{
			"user": username,
			"repo": repoPath,
			"type": objectType,
		}

		switch operation {
		case repo.BlobRead:
			metricBlobReadTotal.With(labels).Inc()
			metricBlobReadBytesTotal.With(labels).Add(float64(payload))
		case repo.BlobWrite:
			metricBlobWriteTotal.With(labels).Inc()
			metricBlobWriteBytesTotal.With(labels).Add(float64(payload))
			metricRepoLastUpdateTimestamp.WithLabelValues(repoPath).Set(
				float64(time.Now().UnixMilli()) / 1000.0)
		case repo.BlobDelete:
			metricBlobDeleteTotal.With(labels).Inc()
			metricBlobDeleteBytesTotal.With(labels).Add(float64(payload))
		case repo.RepoPreloadLastUpdate:
			metricRepoLastUpdateTimestamp.WithLabelValues(repoPath).Set(float64(payload) / 1000.0)
		}
	}
	return f
}

func init() {
	// These are always initialized, but only updated if Config.Prometheus is set
	prometheus.MustRegister(metricBlobWriteTotal)
	prometheus.MustRegister(metricBlobWriteBytesTotal)
	prometheus.MustRegister(metricBlobReadTotal)
	prometheus.MustRegister(metricBlobReadBytesTotal)
	prometheus.MustRegister(metricBlobDeleteTotal)
	prometheus.MustRegister(metricBlobDeleteBytesTotal)
	prometheus.MustRegister(metricRepoLastUpdateTimestamp)
}
