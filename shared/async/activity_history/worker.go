package activity_history

import (
	"database/sql"
	"encoding/json"
	"log"
	"raidhub/shared/async"
	"raidhub/shared/async/bonus_pgcr"
	"raidhub/shared/bungie"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	outgoing *amqp.Channel
	once     sync.Once
)

func create_outbound_channel() {
	once.Do(func() {
		conn, err := async.Init()
		if err != nil {
			log.Fatalf("Failed to create outbound channel: %s", err)
		}
		outgoing, _ = conn.Channel()
	})
}

func process_queue(msgs <-chan amqp.Delivery, db *sql.DB) {
	db.Close() // Not needed
	create_outbound_channel()
	for msg := range msgs {
		process_request(&msg)
	}
}

func process_request(msg *amqp.Delivery) {
	defer func() {
		if err := msg.Ack(false); err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
	}()

	var request ActivityHistoryRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Printf("Failed to unmarshal message: %s", err)
		return
	}

	stats, err := bungie.GetHistoricalStats(request.MembershipType, request.MembershipId)
	if err != nil {
		log.Printf("Failed to get stats: %s", err)
	}

	out := make(chan int64, 2000)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for instanceId := range out {
			bonus_pgcr.SendBonusPGCRMessage(outgoing, instanceId)
		}
		wg.Done()
	}()

	for _, character := range stats.Characters {
		bungie.GetActivityHistory(request.MembershipType, request.MembershipId, character.CharacterId, out)
	}

	close(out)
	wg.Wait()
}
