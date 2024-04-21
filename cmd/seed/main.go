package main

import (
	"log"
	"raidhub/shared/async"
	"raidhub/shared/async/activity_history"

	_ "github.com/lib/pq"
)

const (
	membershipId = 4611686018488107374
	membershipType = 3
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

	activity_history.SendMessage(rabbitChannel, membershipType, membershipId)
	
	log.Println("Queued all players for crawl and activity history. Make sure to run bin/hermes")
}
