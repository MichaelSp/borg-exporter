package app

import (
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

func (a *App) metrics(res http.ResponseWriter, req *http.Request) {
	if a.MetricsMutex.TryLock() {
		defer a.MetricsMutex.Unlock()
	} else {
		slog.Info("metrics request", slog.String("status", "failed"))
		http.Error(res, "metrics request already running", http.StatusServiceUnavailable)
		return
	}
	startTime := time.Now()
	metricRequest := newAppRequest()
	metricRequest.collectMetrics(a.BorgmaticConfigs)
	h := promhttp.HandlerFor(metricRequest.registry, promhttp.HandlerOpts{})
	h.ServeHTTP(res, req)
	slog.Info("metrics request", slog.Duration("duration", time.Since(startTime)))
}

func (req *MetricRequest) collectMetrics(borgmaticConfigs []string) {
	borgmaticConfigsStr := strings.Join(borgmaticConfigs, "-c ")
	slog.Info("get metrics", slog.String("borgmaticConfigsStr", borgmaticConfigsStr))
	if borgmaticConfigsStr != "" {
		borgmaticConfigsStr = "-c " + borgmaticConfigsStr
	}
	repoInfos, err := runBorgmaticCmd[RepoInfos]("borgmatic info " + borgmaticConfigsStr + " --json")
	if err != nil {
		req.errorFetchingRepositoryInfo.With(prometheus.Labels{"error": err.Error()}).Inc()
		slog.Error("Failed to get repo info", slog.Any("error", err))
		return
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
}

func runBorgmaticCmd[T ListArchives | RepoInfos](cmd string) (T, error) {
	slog.Info("Running command", slog.String("cmd", cmd))
	result, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run command '%s': %w", cmd, err)
	}
	var output T
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}
	return output, nil
}
