package server

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/account"
	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/config"
	"github.com/Galang17061/strata-api/internal/web"
)

func New(cfg config.Config, db *sqlx.DB) http.Handler {
	tokens := auth.NewTokenIssuer(cfg.JWTKey, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTExpiryMinutes)
	cipher := auth.NewCipher(cfg.PasswordKey)
	mux := web.NewRouter()
	mux.Use(auth.Authenticate(tokens))
	mux.Get("/health", web.Health)
	account.NewHandler(account.NewService(account.NewStore(db), cipher, tokens)).Mount(mux)
	return mux
}
