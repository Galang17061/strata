package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Galang17061/strata-api/internal/config"
	"github.com/Galang17061/strata-api/internal/database"
	"github.com/Galang17061/strata-api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := migrate(cfg); err != nil {
			log.Fatal(err)
		}
		fmt.Println("schema is in place")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	db, err := database.Open(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(cfg, db),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("Strata listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func migrate(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		if ensureErr := database.EnsureDatabase(ctx, cfg.DatabaseURL, cfg.MigrationsDir); ensureErr != nil {
			return fmt.Errorf("%w (and the database could not be created: %v)", err, ensureErr)
		}
		if db, err = database.Open(ctx, cfg.DatabaseURL); err != nil {
			return err
		}
	}
	defer db.Close()
	return database.Migrate(ctx, db, cfg.MigrationsDir)
}
