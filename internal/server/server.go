package server

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/Galang17061/strata-api/docs"
	"github.com/Galang17061/strata-api/internal/account"
	"github.com/Galang17061/strata-api/internal/audit"
	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/config"
	"github.com/Galang17061/strata-api/internal/feedback"
	"github.com/Galang17061/strata-api/internal/jobs"
	"github.com/Galang17061/strata-api/internal/mail"
	"github.com/Galang17061/strata-api/internal/master"
	"github.com/Galang17061/strata-api/internal/metrics"
	"github.com/Galang17061/strata-api/internal/rbd"
	"github.com/Galang17061/strata-api/internal/web"
)

func New(cfg config.Config, db *sqlx.DB) http.Handler {
	tokens := auth.NewTokenIssuer(cfg.JWTKey, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTExpiryMinutes)
	cipher := auth.NewCipher(cfg.PasswordKey)
	mux := web.NewRouter()
	pulse := metrics.NewCollector()
	mux.Use(pulse.Middleware)
	mux.Use(auth.Authenticate(tokens))
	auditStore := audit.NewStore(db)
	mux.Use(audit.Middleware(auditStore))
	mux.Get("/metrics", pulse.Handler)
	mux.Get("/health", web.Health)
	mux.Get("/files/*", web.StaticFiles(cfg.UploadDir))
	mux.Get("/swagger/*", httpSwagger.WrapHandler)
	mailer := mail.NewSender(cfg.MailHost, cfg.MailPort, cfg.MailUsername, cfg.MailPassword, cfg.MailFrom)
	account.NewHandler(account.NewService(account.NewStore(db), cipher, tokens, mailer, cfg.WebURL)).Mount(mux)
	master.NewHandler(master.NewService(master.NewStore(db), cfg.UploadDir)).Mount(mux)
	rbdStore := rbd.NewStore(db)
	totalService := rbd.NewTotalService(rbdStore)
	totals := rbd.NewTotalHandler(totalService)
	rbd.NewPlotHandler(rbd.NewPlotService(rbdStore, totalService)).Mount(mux)
	rbd.NewSystemHandler(rbd.NewSystemService(rbdStore)).Mount(mux)
	snapshots := rbd.NewSnapshotService(rbdStore)
	rbd.NewHierarchyHandler(rbd.NewHierarchyService(rbdStore).WithSnapshots(snapshots)).Mount(mux)
	rbd.NewComponentHandler(rbd.NewComponentService(rbdStore).WithSnapshots(snapshots)).Mount(mux)
	rbd.NewEditorHandler(rbd.NewDrawingService(rbdStore)).WithAfterEdgeSave(totals.AfterEdgeSave).Mount(mux)
	parameters := rbd.NewParameterService(rbdStore)
	rbd.NewFailureHandler(rbd.NewFailureService(rbdStore), parameters).Mount(mux)
	rbd.NewWeibullHandler(parameters).Mount(mux)
	rbd.NewPoissonHandler(parameters).Mount(mux)
	gate := rbd.NewGate(cfg.GAConcurrency, time.Duration(cfg.GAQueueWait)*time.Second)
	rbd.NewOptimizationHandler(rbd.NewOptimizationService(rbdStore).WithGate(gate)).Mount(mux)
	queue := jobs.NewStore(db)
	listener := jobs.NewListener(cfg.DatabaseURL)
	listener.Start(context.Background())
	jobs.NewHandler(queue, listener).Mount(mux)
	rbd.NewSimulationHandler(rbd.NewSimulationService(rbdStore), queue).Mount(mux)
	rbd.NewSnapshotHandler(snapshots).Mount(mux)
	audit.NewHandler(auditStore).Mount(mux)
	feedback.NewHandler(feedback.NewStore(db), mailer, cfg.FeedbackEmail).Mount(mux)
	rbd.NewNotifyHandler(rbdStore).Mount(mux)
	rbd.NewBatchHandler(rbd.NewBatchService(rbdStore, parameters, totalService)).Mount(mux)
	totals.Mount(mux)
	return mux
}
