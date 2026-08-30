package server

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/account"
	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/config"
	"github.com/Galang17061/strata-api/internal/master"
	"github.com/Galang17061/strata-api/internal/rbd"
	"github.com/Galang17061/strata-api/internal/web"
)

func New(cfg config.Config, db *sqlx.DB) http.Handler {
	tokens := auth.NewTokenIssuer(cfg.JWTKey, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTExpiryMinutes)
	cipher := auth.NewCipher(cfg.PasswordKey)
	mux := web.NewRouter()
	mux.Use(auth.Authenticate(tokens))
	mux.Get("/health", web.Health)
	mux.Get("/files/*", web.StaticFiles(cfg.UploadDir))
	account.NewHandler(account.NewService(account.NewStore(db), cipher, tokens)).Mount(mux)
	master.NewHandler(master.NewService(master.NewStore(db), cfg.UploadDir)).Mount(mux)
	rbdStore := rbd.NewStore(db)
	totalService := rbd.NewTotalService(rbdStore)
	totals := rbd.NewTotalHandler(totalService)
	rbd.NewPlotHandler(rbd.NewPlotService(rbdStore, totalService)).Mount(mux)
	rbd.NewSystemHandler(rbd.NewSystemService(rbdStore)).Mount(mux)
	rbd.NewHierarchyHandler(rbd.NewHierarchyService(rbdStore)).Mount(mux)
	rbd.NewComponentHandler(rbd.NewComponentService(rbdStore)).Mount(mux)
	rbd.NewEditorHandler(rbd.NewDrawingService(rbdStore)).WithAfterEdgeSave(totals.AfterEdgeSave).Mount(mux)
	parameters := rbd.NewParameterService(rbdStore)
	rbd.NewFailureHandler(rbd.NewFailureService(rbdStore), parameters).Mount(mux)
	rbd.NewWeibullHandler(parameters).Mount(mux)
	totals.Mount(mux)
	return mux
}
