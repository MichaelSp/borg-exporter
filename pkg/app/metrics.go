package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricsCollectionTimeout = 14 * time.Minute

func (a *App) metrics(res http.ResponseWriter, req *http.Request) {
	if a.MetricsMutex.TryLock() {
		defer a.MetricsMutex.Unlock()
	} else {
		slog.Info("metrics request", slog.String("status", "failed"))
		http.Error(res, "metrics request already running", http.StatusServiceUnavailable)
		return
	}
	// Borgmatic may inspect several repositories serially. Keep this below the
	// Prometheus scrape timeout while allowing normal repository latency.
	ctx, cancel := context.WithTimeout(req.Context(), metricsCollectionTimeout)
	defer cancel()

	startTime := time.Now()
	metricRequest := newAppRequest()
	if err := a.RepoLock.With(ctx, func(ctx context.Context) error {
		return metricRequest.collectMetrics(ctx, a.BorgmaticConfigs)
	}); err == nil {
		snapshot := MetricsSnapshot{Metrics: renderMetrics(metricRequest.registry), SavedAt: time.Now()}
		a.saveSnapshot(snapshot)
		writeSnapshot(res, snapshot)
		slog.Info("metrics request", slog.Duration("duration", time.Since(startTime)))
		return
	} else {
		slog.Info("metrics refresh failed", slog.Any("error", err))
		if snapshot, found := a.loadSnapshot(req.Context()); found {
			writeSnapshot(res, snapshot)
			return
		}
		http.Error(res, "Borg repository is busy and no cached metrics are available", http.StatusServiceUnavailable)
	}
}

func (a *App) saveSnapshot(snapshot MetricsSnapshot) {
	if a.MetricsCache == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.MetricsCache.Save(ctx, snapshot); err != nil {
		slog.Warn("failed to save metrics cache", slog.Any("error", err))
	}
}

func (a *App) loadSnapshot(ctx context.Context) (MetricsSnapshot, bool) {
	if a.MetricsCache == nil {
		return MetricsSnapshot{}, false
	}
	snapshot, found, err := a.MetricsCache.Load(ctx)
	if err != nil {
		slog.Warn("failed to load metrics cache", slog.Any("error", err))
		return MetricsSnapshot{}, false
	}
	return snapshot, found
}

func renderMetrics(registry *prometheus.Registry) string {
	response := &metricsResponse{header: make(http.Header)}
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(response, nil)
	return response.body.String()
}

func writeSnapshot(res http.ResponseWriter, snapshot MetricsSnapshot) {
	res.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = res.Write([]byte(snapshot.Metrics))
	age := time.Since(snapshot.SavedAt).Seconds()
	if age < 0 {
		age = 0
	}
	_, _ = fmt.Fprintf(res, "# HELP borg_exporter_snapshot_last_success_timestamp_seconds Unix timestamp of the last successful repository metrics refresh.\n# TYPE borg_exporter_snapshot_last_success_timestamp_seconds gauge\nborg_exporter_snapshot_last_success_timestamp_seconds %.3f\n# HELP borg_exporter_snapshot_age_seconds Age of the last successful repository metrics snapshot.\n# TYPE borg_exporter_snapshot_age_seconds gauge\nborg_exporter_snapshot_age_seconds %.3f\n", float64(snapshot.SavedAt.UnixNano())/float64(time.Second), age)
}

type metricsResponse struct {
	header http.Header
	body   bytes.Buffer
}

func (r *metricsResponse) Header() http.Header         { return r.header }
func (r *metricsResponse) WriteHeader(statusCode int)  {}
func (r *metricsResponse) Write(b []byte) (int, error) { return r.body.Write(b) }

func (req *MetricRequest) collectMetrics(ctx context.Context, borgmaticConfigs []string) error {
	borgmaticConfigsStr := strings.Join(borgmaticConfigs, "-c ")
	slog.Info("get metrics", slog.String("borgmaticConfigsStr", borgmaticConfigsStr))
	if borgmaticConfigsStr != "" {
		borgmaticConfigsStr = "-c " + borgmaticConfigsStr
	}
	repoInfos, err := runBorgmaticCmd[RepoInfos](ctx, "borgmatic info "+borgmaticConfigsStr+" --json")
	if err != nil {
		req.errorFetchingRepositoryInfo.With(prometheus.Labels{"error": err.Error()}).Inc()
		slog.Error("Failed to get repo info", slog.Any("error", err))
		return err
	}

	for i := range repoInfos {
		repoInfo := repoInfos[i]
		archives := repoInfo.Archives
		labels := prometheus.Labels{
			"location":  repoInfo.Repository.Location,
			"repoLabel": repoInfo.Repository.Label,
			"archive":   "",
		}

		if len(archives) == 0 {
			req.totalSize.With(labels).Set(float64(len(archives)))
			continue
		}

		latestArchive := archives[len(archives)-1]
		latestArchiveTime, err := time.Parse("2006-01-02T15:04:05.000000", latestArchive.Start)
		if err != nil {
			req.errorFetchingRepositoryInfo.With(prometheus.Labels{"error": err.Error()}).Inc()
			slog.Error("Failed to parse time", slog.Any("error", err))
			continue
		}
		archiveName := latestArchive.Name
		// remove trailing unix timestamp from archive name.
		archiveName = regexp.MustCompile(`-\d{10}$`).ReplaceAllString(archiveName, "")
		labels["archive"] = latestArchive.Name
		unixTimestamp := latestArchiveTime.Unix()
		req.lastBackupTimestamp.With(labels).Set(float64(unixTimestamp))
		req.uniqueSize.With(labels).Set(float64(latestArchive.Stats.OriginalSize))
		req.numberOfFiles.With(labels).Set(float64(latestArchive.Stats.Nfiles))
		req.totalSize.With(labels).Set(float64(len(archives)))
		req.compressedSize.With(labels).Set(float64(latestArchive.Stats.CompressedSize))
		req.deduplicatedSize.With(labels).Set(float64(latestArchive.Stats.DeduplicatedSize))
		req.cacheSize.With(labels).Set(float64(repoInfo.Cache.Stats.TotalSize))
	}

	return nil
}

func runBorgmaticCmd[T ListArchives | RepoInfos](ctx context.Context, cmd string) (T, error) {
	slog.Info("Running command", slog.String("cmd", cmd))
	result, err := exec.CommandContext(ctx, "sh", "-c", cmd).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run command '%s': %w", cmd, err)
	}
	var output T
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}
	return output, nil
}
