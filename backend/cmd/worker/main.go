package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"imageprocessor/backend/internal/config"
	"imageprocessor/backend/internal/processor"
	"imageprocessor/backend/internal/queue"
	"imageprocessor/backend/internal/storage"
)

func main() {
	cfg := config.Load()

	store, err := storage.NewFileStore(cfg.StoragePath)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	imageProcessor := processor.New(store)
	consumer := queue.NewKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	defer consumer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("worker started")
	if err := consumer.Consume(ctx, imageProcessor.Handle); err != nil && ctx.Err() == nil {
		log.Fatalf("worker consume: %v", err)
	}
}
