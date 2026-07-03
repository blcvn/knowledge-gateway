package cmd

import (
	"kg-service/internal/bootstrap"
	"kg-service/internal/config"
)

var serveCmd = &Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start kg-service with the configured HTTP, Postgres, Redis, and bootstrap settings.",
	run: func(args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		app, err := bootstrap.New(cfg)
		if err != nil {
			return err
		}

		return app.Run()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
	RootCmd.run = serveCmd.run
}
