package app

import (
	"fmt"
	"log/slog"
	"net/http"
)

func (a *App) Run() error {
	http.HandleFunc("/", a.index)
	http.HandleFunc("/metrics", a.metrics)

	slog.Info("Started borg-exporter", slog.String("PORT", a.Port))
	err := http.ListenAndServe(fmt.Sprintf(":%s", a.Port), nil)
	if err != nil {
		slog.Error("Failed to start server: %v", err)
		return err
	}
	return nil
}
