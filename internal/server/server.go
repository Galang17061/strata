package server

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/config"
	"github.com/Galang17061/strata-api/internal/web"
)

func New(cfg config.Config, db *sqlx.DB) http.Handler {
	mux := web.NewRouter()
	mux.Get("/health", web.Health)
	return mux
}
