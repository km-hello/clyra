package httpapi

import (
	"database/sql"
	"net/http"

	"clyra/backend/internal/config"
	"clyra/backend/internal/integrations"
	"clyra/backend/internal/notification"
	"clyra/backend/internal/subscription"
)

type API struct {
	db            *sql.DB
	config        config.Config
	HTTPClient    *http.Client
	notifications *notification.Service
	subscriptions subscription.Service
	iconCatalogs  *integrations.CatalogSearch
}

func New(db *sql.DB, cfg config.Config, client *http.Client, notifications *notification.Service) *API {
	return &API{db: db, config: cfg, HTTPClient: client, notifications: notifications, subscriptions: subscription.Service{DB: db}, iconCatalogs: integrations.NewCatalogSearch()}
}
