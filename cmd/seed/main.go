package main

import (
	"flag"
	"log"
	"raidhub/shared/async"
	"raidhub/shared/async/activity_history"
	"raidhub/shared/async/player_crawl"
	"raidhub/shared/postgres"
	"strconv"
	"strings"
	"sync"
	"time"

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

	db, err := postgres.Connect()
	if err != nil {
		log.Fatalf("Error connecting to the database: %s", err)
	}

	var wg sync.WaitGroup
	for _, id := range idsArr {
		id, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			log.Fatalf("Failed to parse membership id: %s", err)
		}
		player_crawl.SendPlayerCrawlMessage(rabbitChannel, id)

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(3 * time.Second)
			var membershipType int
			var membershipId int64

			err := db.QueryRow("SELECT membership_type, membership_id FROM player WHERE membership_id = $1::bigint", id).Scan(&membershipType, &membershipId)
			if err != nil {
				log.Printf("Failed to get player: %s", err)
			}

			activity_history.SendActivityHistoryRequest(rabbitChannel, membershipType, membershipId)
		}()
	}

	wg.Wait()
	log.Println("Queued all players for crawl and activity history. Make sure to run bin/hermes")
}
