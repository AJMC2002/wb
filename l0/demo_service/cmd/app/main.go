package main

import (
	"context"
	"log"
	"time"

	httpin "demo_service/internal/adapters/inbound/http"
	kafkain "demo_service/internal/adapters/inbound/kafka"
	"demo_service/internal/adapters/outbound/cache"
	"demo_service/internal/adapters/outbound/postgres"
	"demo_service/internal/app/config"
	"demo_service/internal/app/runtime"
	"demo_service/internal/core/service"
)

func main() {
	ctx, stop := runtime.NotifyContext(context.Background())
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.Close()

	// migrations
	migCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := postgres.RunMigrations(migCtx, db.Pool, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	repo := postgres.NewOrderRepository(db.Pool)
	memCache := cache.NewMemoryCache()
	svc := service.NewOrderService(repo, memCache)

	// warm cache
	if n, err := svc.WarmCache(ctx, cfg.CacheWarmLimit); err != nil {
		log.Printf("[warmup] failed: %v", err)
	} else {
		log.Printf("[warmup] cache loaded: %d orders (cache size=%d)", n, memCache.Len(ctx))
	}

	// HTTP
	handlers := httpin.NewHandlers(svc)
	mux := httpin.NewMux(handlers, svc)
	httpSrv := runtime.NewHTTPServer(cfg.HTTPAddr, mux)
	httpSrv.Start()

	// kafka consumer
	consumer := kafkain.NewConsumer(kafkain.ConsumerConfig{
		Brokers:  cfg.KafkaBrokers,
		Topic:    cfg.KafkaTopic,
		GroupID:  cfg.KafkaConsumerGroup,
		MinBytes: cfg.KafkaMinBytes,
		MaxBytes: cfg.KafkaMaxBytes,
	}, svc)
	defer func() { _ = consumer.Close() }()

	go consumer.Run(ctx)

	<-ctx.Done()
	log.Printf("[shutdown] signal received")

	if err := httpSrv.Shutdown(context.Background(), cfg.ShutdownTimeout); err != nil {
		log.Printf("[shutdown] http: %v", err)
	}
	log.Printf("[shutdown] bye")
}
