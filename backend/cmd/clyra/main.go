package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"clyra/backend/internal/app"
	"clyra/backend/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	service, err := app.New(cfg)
	if err != nil {
		return err
	}
	defer service.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("Clyra listening on %s", cfg.Addr)
	return service.Run(ctx)
}
