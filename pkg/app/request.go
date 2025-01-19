package app

import (
	"github.com/prometheus/client_golang/prometheus"
)

type MetricRequest struct {
	registry                    *prometheus.Registry
	totalSize                   *prometheus.GaugeVec
	lastBackupTimestamp         *prometheus.GaugeVec
	uniqueSize                  *prometheus.GaugeVec
	numberOfFiles               *prometheus.GaugeVec
	errorFetchingRepositoryInfo *prometheus.GaugeVec
}

func newAppRequest() MetricRequest {
	labels := []string{"location", "repoLabel", "archive"}
	req := MetricRequest{
		registry: prometheus.NewRegistry(),
		totalSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_backups_total",
				Help: "Total number of Borg backups",
			},
			labels,
		),
		lastBackupTimestamp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_last_backup_timestamp",
				Help: "Timestamp of the last backup",
			},
			labels,
		),
		uniqueSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_unique_size",
				Help: "Uncompressed unique size of the Borg Repo (bytes)",
			},
			labels,
		),
		numberOfFiles: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_number_of_files",
				Help: "Number of files in the Borg Repo",
			},
			labels,
		),
		errorFetchingRepositoryInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_error_fetching_repository_info",
				Help: "Error fetching repository info",
			},
			labels,
		),
	}
	req.registry.MustRegister(req.totalSize)
	req.registry.MustRegister(req.lastBackupTimestamp)
	req.registry.MustRegister(req.uniqueSize)
	req.registry.MustRegister(req.numberOfFiles)
	req.registry.MustRegister(req.errorFetchingRepositoryInfo)
	return req
}
