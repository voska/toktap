package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voska/toktap/internal/config"
	"github.com/voska/toktap/internal/influx"
	"github.com/voska/toktap/internal/pricing"
	"github.com/voska/toktap/internal/proxy"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("toktap %s (commit=%s, built=%s)\n", version, commit, date)
		os.Exit(0)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	routes, err := proxy.LoadRoutes(cfg.RoutesPath)
	if err != nil {
		log.Fatalf("loading routes: %v", err)
	}

	pricingTable, err := pricing.LoadFromFile(cfg.PricingPath)
	if err != nil {
		log.Printf("warning: could not load pricing config: %v (cost calculation disabled)", err)
	}

	if pricingTable != nil {
		go func() {
			for range time.Tick(60 * time.Second) {
				if err := pricingTable.Reload(cfg.PricingPath); err != nil {
					log.Printf("pricing reload error: %v", err)
				}
			}
		}()
	}

	writer := influx.NewWriter(cfg.InfluxURL, cfg.InfluxToken, cfg.InfluxOrg, cfg.InfluxBucket)
	defer writer.Close()

	p := proxy.New(routes, writer, pricingTable)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           p,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("toktap %s listening on :%s", version, cfg.Port)
	for name, route := range routes {
		log.Printf("  /%s → %s (provider=%s)", name, route.Upstream, route.Provider)
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	log.Printf("shutting down (signal=%s), draining connections...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Print("shutdown complete")
}
