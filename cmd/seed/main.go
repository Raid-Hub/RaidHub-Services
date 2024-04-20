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
		log.Fatalf("Failed to create rabbit connection: %s", err)
	}
	defer async.Cleanup()

	rabbitChannel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create rabbit channel: %s", err)
	}
	defer rabbitChannel.Close()

	var wg sync.WaitGroup
	activity_history.SendMessage(rabbitChannel, 3, 4611686018488107374)

	wg.Wait()
	log.Println("Queued all players for crawl and activity history. Make sure to run bin/hermes")
}
