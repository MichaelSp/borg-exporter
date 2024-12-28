package app

import (
	"encoding/json"
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
	archiveList := runBorgmaticCmd[ListArchives]("borgmatic list " + borgmaticConfigsStr + " --json")
	repoInfos := runBorgmaticCmd[RepoInfos]("borgmatic info " + borgmaticConfigsStr + " --json")

	if archiveList == nil || repoInfos == nil {
		slog.Error("Failed to get archive list or repo info")
		return
	}

	if len(archiveList) != len(repoInfos) {
		slog.Error("Archive list and repo info have different lengths")
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
	}
}

func runBorgmaticCmd[T ListArchives | RepoInfos](cmd string) T {
	slog.Info("Running command", slog.String("cmd", cmd))
	result, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		slog.Error("Command failed", slog.Any("error", err))
	}
	var output T
	if err := json.Unmarshal(result, &output); err != nil {
		slog.Error("Failed to unmarshal JSON", slog.Any("error", err))
	}
	return output
}
