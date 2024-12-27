package main

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/michaelsp/borg-exporter/pkg/app"
)

func main() {
	borgmaticConfigs := os.Getenv("BORGMATIC_CONFIG")
	port := os.Getenv("PORT")
	_, err := strconv.Atoi(port)
	if port == "" || err != nil {
		port = "9996"
	}

	a := app.App{
		BorgmaticConfigs: strings.Split(borgmaticConfigs, ","),
		Port:             port,
	}
	err = a.Run()
	if err != nil {
		slog.Error("Failed to run app: %v", err)
	}
}
