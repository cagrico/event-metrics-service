package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	eventsHttp "event-metrics-service/internal/events/adapters/http/fiber"
	eventsRepoPg "event-metrics-service/internal/events/adapters/postgres"
	eventsUsecase "event-metrics-service/internal/events/core/usecase"

	metricsHttp "event-metrics-service/internal/metrics/adapters/http/fiber"
	metricsRepoPg "event-metrics-service/internal/metrics/adapters/postgres"
	metricsUsecase "event-metrics-service/internal/metrics/core/usecase"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "event-metrics-service/docs"
)

func main() {
	// Config
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("POSTGRES_DSN is not set")
	}

	// DB connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open postgres: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	// Adapter-level DB wrappers
	eventsDB := eventsRepoPg.NewSQLDB(db)
	metricsDB := metricsRepoPg.NewSQLDB(db)

	// Repositories
	eventRepository := eventsRepoPg.NewEventRepository(eventsDB)
	metricsRepository := metricsRepoPg.NewMetricsRepository(metricsDB)

	// Usecaseses
	storeEventUC := eventsUsecase.NewStoreEventUseCase(eventRepository)
	getMetricsUC := metricsUsecase.NewGetMetricsUseCase(metricsRepository)

	// HTTP (Fiber) app + handlers
	app := fiber.New()

	// events endpoints
	eventsHandler := eventsHttp.NewEventHandler(storeEventUC)
	app.Post("/events", eventsHandler.CreateEvent)
	app.Post("/events/bulk", eventsHandler.BulkCreateEvents)

	// metrics endpoints
	metricsHandler := metricsHttp.NewMetricsHandler(getMetricsUC)
	app.Get("/metrics", metricsHandler.GetMetrics)

	// Swagger
	app.Get("/docs/*", fiberSwagger.WrapHandler)

	// Graceful shutdown
	go func() {
		if err := app.Listen(":8080"); err != nil {
			log.Printf("fiber stopped: %v", err)
		}
	}()

	log.Println("server started on :8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("fiber shutdown error: %v", err)
	}

	log.Println("server exiting")
}
