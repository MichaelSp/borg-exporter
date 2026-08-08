package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/michaelsp/borg-exporter/pkg/app"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "lock-run" {
		if err := lockRun(os.Args[2:]); err != nil {
			slog.Error("repository-locked command failed", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	borgmaticConfigs := os.Getenv("BORGMATIC_CONFIG")
	port := os.Getenv("PORT")
	_, err := strconv.Atoi(port)
	if port == "" || err != nil {
		port = "9996"
	}

	metricsCache, err := app.NewConfigMapMetricsCacheFromEnv()
	if err != nil {
		slog.Error("failed to configure metrics cache", slog.Any("error", err))
		return
	}
	a := newApp(borgmaticConfigs, port, metricsCache)
	err = a.Run()
	if err != nil {
		slog.Error("failed to run app", slog.Any("error", err))
	}
}

func newApp(borgmaticConfigs, port string, metricsCache app.MetricsCache) app.App {
	return app.App{
		BorgmaticConfigs: strings.Split(borgmaticConfigs, ","),
		Port:             port,
		MetricsMutex:     sync.Mutex{},
		RepoLock:         app.NewRepoLockFromEnv(),
		MetricsCache:     metricsCache,
	}
}

func lockRun(args []string) error {
	if len(args) < 2 || args[0] != "--" {
		return fmt.Errorf("usage: borg-exporter lock-run -- <command> [args...]")
	}

	return app.NewRepoLockFromEnv().With(context.Background(), func(ctx context.Context) error {
		command := exec.CommandContext(ctx, args[1], args[2:]...)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		return command.Run()
	})
}
