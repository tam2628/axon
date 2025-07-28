package common

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type App struct {
	Logger       *zap.Logger
	Router       *httprouter.Router
	PromRegistry *prometheus.Registry
}

func InitApp() App {
	logger, err := InitLogger()

	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	defer logger.Sync()

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	router := httprouter.New()

	router.Handler("GET", "/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	App := App{
		Logger:       logger,
		Router:       router,
		PromRegistry: registry,
	}

	return App
}

func (app App) RunServerWithGracefulShutdown(port uint16) error {

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: app.Router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		app.Logger.Info("Starting server", zap.Uint16("port", port))
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				// Log the error if it's not a graceful shutdown
				app.Logger.Error("Server failed", zap.Error(err))
				return
			}
		}
	}()

	<-quit
	app.Logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		app.Logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	app.Logger.Info("Server exited gracefully")
	return nil
}
