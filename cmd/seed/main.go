package main

import (
	"log"
	"raidhub/shared/async"
	"raidhub/shared/async/activity_history"
	"sync"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to the RabbitMQ
	conn, err := async.Init()
	if err != nil {
		log.Fatalf("Failed to create connection: %s", err)
	}
	defer async.Cleanup()

	rabbitChannel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %s", err)
	}
	defer rabbitChannel.Close()

	var wg sync.WaitGroup
	if err != nil {
		log.Fatalf("Failed to parse membership id: %s", err)
	}

	activity_history.SendMessage(rabbitChannel, 3, 4611686018488107374)

	wg.Wait()
	log.Println("Queued all players for crawl and activity history. Make sure to run bin/hermes")
}
