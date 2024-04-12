package main

import (
	"flag"
	"log"
	"raidhub/shared/async"
	"raidhub/shared/async/activity_history"
	"raidhub/shared/async/player_crawl"
	"strconv"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

func main() {
	// Define a flag for the BungieNames
	ids := flag.String("ids", "", "Comma-separated list of membershipIds")

	flag.Parse()

	if *ids == "" {
		log.Fatal("membership ids are required")
	}

	// Split the comma-separated BungieNames into a slice
	idsArr := strings.Split(*ids, ",")

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
	for _, id := range idsArr {
		id, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			log.Fatalf("Failed to parse membership id: %s", err)
		}

		go player_crawl.SendPlayerCrawlMessage(rabbitChannel, id)

		wg.Add(1)
		go func() {
			defer wg.Done()
			activity_history.SendActivityHistoryRequest(rabbitChannel, 3, 4611686018488107374)
		}()
	}

	wg.Wait()
	log.Println("Queued all players for crawl and activity history. Make sure to run bin/hermes")
}
