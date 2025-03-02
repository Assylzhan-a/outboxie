package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/assylzhan-a/outboxie/internal/example/handler"
	"github.com/assylzhan-a/outboxie/internal/example/repository"
	"github.com/assylzhan-a/outboxie/internal/example/service"
	"github.com/assylzhan-a/outboxie/pkg/outbox"
	"github.com/assylzhan-a/outboxie/pkg/outbox/config"
)

type Config struct {
	InstanceID string
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	NatsURL    string
	HTTPPort   int
}

type App struct {
	config        Config
	dbPool        *pgxpool.Pool
	natsConn      *nats.Conn
	outboxService *outbox.Outbox
	server        *http.Server
}

func New(cfg Config) *App {
	return &App{
		config: cfg,
	}
}

func (a *App) Setup(ctx context.Context) error {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		a.config.DBUser,
		a.config.DBPassword,
		a.config.DBHost,
		a.config.DBPort,
		a.config.DBName,
	)

	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	a.dbPool = dbPool

	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}
	log.Println("Connected to PostgreSQL")

	// Create the outbox configuration
	outboxConfig := config.NewOutboxConfig(dbPool, a.config.NatsURL, a.config.InstanceID).
		WithPollingInterval(100 * time.Millisecond).
		WithBatchSize(10).
		WithMaxRetries(3)

	// Create the outbox
	outboxService, err := outbox.New(outboxConfig)
	if err != nil {
		return fmt.Errorf("failed to create outbox: %w", err)
	}
	a.outboxService = outboxService

	// to verify the connection
	nc, err := nats.Connect(a.config.NatsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	a.natsConn = nc
	log.Println("Connected to NATS")

	mux := http.NewServeMux()

	orderRepo := repository.NewPostgresOrderRepository(dbPool)

	orderService := service.NewOrderService(dbPool, orderRepo, outboxService)

	orderHandler := handler.NewOrderHandler(orderService)

	mux.HandleFunc("/orders", orderHandler.CreateOrder)

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.HTTPPort),
		Handler: mux,
	}

	return nil
}

func (a *App) Start(ctx context.Context) error {
	// Start the outbox processor
	if err := a.outboxService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start outbox processor: %w", err)
	}

	log.Printf("Starting HTTP server on port %d", a.config.HTTPPort)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			log.Printf("Failed to shut down HTTP server: %v", err)
		}
	}

	// Stop the outbox processor
	if a.outboxService != nil {
		if err := a.outboxService.Stop(); err != nil {
			log.Printf("Failed to stop outbox processor: %v", err)
		}
	}

	if a.natsConn != nil {
		a.natsConn.Close()
	}

	if a.dbPool != nil {
		a.dbPool.Close()
	}

	return nil
}
