package app

import (
	"log/slog"
	"net/http"
)

func (a *App) index(res http.ResponseWriter, _ *http.Request) {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.WriteHeader(http.StatusOK)
	_, err := res.Write([]byte(`
<!doctype html>
<html lang="en">
  <head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <title>borg-exporter</title>
  </head>
  <body>
    <h1>Borg Exporter</h1>
    <p><a href="/metrics">Metrics</a></p>
  </body>
</html>
`))
	if err != nil {
		slog.Warn("Failed to write response", slog.Any("error", err))
	}
}
