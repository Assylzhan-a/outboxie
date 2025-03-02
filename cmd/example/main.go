package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assylzhan-a/outboxie/internal/example/app"
)

func main() {
	instanceID := flag.String("instance-id", "instance1", "Unique identifier for this instance")
	dbHost := flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort := flag.Int("db-port", 5433, "PostgreSQL port")
	dbUser := flag.String("db-user", "postgres", "PostgreSQL user")
	dbPassword := flag.String("db-password", "postgres", "PostgreSQL password")
	dbName := flag.String("db-name", "outboxie", "PostgreSQL db name")
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS URL")
	httpPort := flag.Int("http-port", 8080, "HTTP port")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	application := app.New(app.Config{
		InstanceID: *instanceID,
		DBHost:     *dbHost,
		DBPort:     *dbPort,
		DBUser:     *dbUser,
		DBPassword: *dbPassword,
		DBName:     *dbName,
		NatsURL:    *natsURL,
		HTTPPort:   *httpPort,
	})

	if err := application.Setup(ctx); err != nil {
		log.Fatalf("Failed to set up application: %v", err)
	}

	if err := application.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	//Graceful shutdown
	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := application.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shut down application: %v", err)
	}

	log.Println("Exiting")
}
