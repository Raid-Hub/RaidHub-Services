package main

import (
	"log"
	"os"
	"raidhub/shared/async"
	"raidhub/shared/async/activity_history"
	"raidhub/shared/async/player_crawl"
	"strconv"
	"sync"

	_ "github.com/lib/pq"
)

func main() {
	// Parse the ids
	membershipIds := os.Args[1:]

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
	for _, id := range membershipIds {
		wg.Add(1)
		go func(membershipId string) {
			defer wg.Done()
			idInt64, err := strconv.ParseInt(membershipId, 10, 64)
			if err != nil {
				log.Fatal(err)
			}
			err = player_crawl.SendMessage(rabbitChannel, idInt64)
			if err != nil {
				log.Fatal(err)
			}
			err = activity_history.SendMessage(rabbitChannel, idInt64)
			if err != nil {
				log.Fatal(err)
			}
		}(id)
	}

	wg.Wait()

	log.Println("Queued all players for crawl and activity history. Make sure to run bin/hermes")
}
