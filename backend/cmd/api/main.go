package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"imageprocessor/backend/internal/config"
	"imageprocessor/backend/internal/httpapi"
	"imageprocessor/backend/internal/queue"
	"imageprocessor/backend/internal/storage"
)

func main() {
	cfg := config.Load()

	store, err := storage.NewFileStore(cfg.StoragePath)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	producer := queue.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer producer.Close()

	handler := httpapi.NewHandler(store, producer)
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      handler.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("api server: %v", err)
		}
	}()

	waitForShutdown(server)
}

func waitForShutdown(server *http.Server) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}
