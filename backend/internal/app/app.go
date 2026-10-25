package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	"clyra/backend/internal/config"
	"clyra/backend/internal/database"
	"clyra/backend/internal/httpapi"
	"clyra/backend/internal/notification"
)

type App struct {
	db                  *sql.DB
	config              config.Config
	handler             http.Handler
	notifications       *notification.Service
	cancelNotifications context.CancelFunc
}

func New(cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	notificationCtx, cancelNotifications := context.WithCancel(context.Background())
	notifications := &notification.Service{DB: db, Config: cfg, HTTPClient: client, Context: notificationCtx}
	api := httpapi.New(db, cfg, client, notifications)
	return &App{db: db, config: cfg, handler: api.Handler(), notifications: notifications, cancelNotifications: cancelNotifications}, nil
}
func (a *App) Close() error {
	a.cancelNotifications()
	return a.db.Close()
}
func (a *App) Run(ctx context.Context) error {
	defer a.cancelNotifications()
	var workers sync.WaitGroup
	workers.Add(1)
	go func() { defer workers.Done(); a.notifications.Run(a.notifications.Context) }()
	server := &http.Server{Addr: a.config.Addr, Handler: a.handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 * 1024}
	stopped := make(chan error, 1)
	go func() { stopped <- server.ListenAndServe() }()
	var err error
	select {
	case <-ctx.Done():
	case err = <-stopped:
	}
	a.cancelNotifications()
	// Canceled sends persist their outcome before handlers and the scheduler exit.
	shutdownErr := server.Shutdown(context.Background())
	workers.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return shutdownErr
}
