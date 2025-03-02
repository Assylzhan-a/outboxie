package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS URL")
	subscriberID := flag.String("subscriber-id", "subscriber1", "Unique identifier for this subscriber")
	flag.Parse()

	log.Printf("[Subscriber %s] Starting...", *subscriberID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("[Subscriber %s] Received signal: %v", *subscriberID, sig)
		cancel()
	}()

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("[Subscriber %s] Failed to connect to NATS: %v", *subscriberID, err)
	}
	defer nc.Close()

	log.Printf("[Subscriber %s] Connected to NATS at %s", *subscriberID, *natsURL)

	sub, err := nc.Subscribe("orders.created", func(msg *nats.Msg) {
		log.Printf("[Subscriber %s] Received order created event: %s", *subscriberID, string(msg.Data))
	})
	if err != nil {
		log.Fatalf("[Subscriber %s] Failed to subscribe to orders.created: %v", *subscriberID, err)
	}
	defer sub.Unsubscribe()

	log.Printf("[Subscriber %s] Subscribed to orders.created topic", *subscriberID)
	log.Printf("[Subscriber %s] Waiting for messages...", *subscriberID)

	// Graceful shutdown
	<-ctx.Done()

	log.Printf("[Subscriber %s] Shutting down...", *subscriberID)
	nc.Drain()
	time.Sleep(100 * time.Millisecond)

	log.Printf("[Subscriber %s] Exiting", *subscriberID)
}
