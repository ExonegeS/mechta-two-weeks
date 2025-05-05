package main

import (
	"os"

	"github.com/ExonegeS/mechta-two-weeks/config"
	"github.com/ExonegeS/mechta-two-weeks/internal/app"
	"github.com/ExonegeS/mechta-two-weeks/prettyslog"
)

func main() {
	cfg := config.NewConfig()
	logger := prettyslog.SetupPrettySlog(os.Stdout)

	server := app.NewAPIServer(cfg, logger)
	server.Run()
}
