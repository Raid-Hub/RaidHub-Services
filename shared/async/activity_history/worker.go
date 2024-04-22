package activity_history

import (
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

func process_queue(qw *async.QueueWorker, msgs <-chan amqp.Delivery) {
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
		log.Fatalf("Failed to unmarshal message: %s", err)
		return
	}

	profiles, err := bungie.GetLinkedProfiles(-1, request.MembershipId, false)

	var membershipType int
	for _, profile := range profiles {
		if profile.MembershipId == request.MembershipId {
			membershipType = profile.MembershipType
			break
		}
	}

	if membershipType == 0 {
		log.Printf("Failed to find membership type for %s", request.MembershipId)
		return
	}

	stats, err := bungie.GetHistoricalStats(membershipType, request.MembershipId)
	if err != nil {
		log.Printf("Failed to get stats: %s", err)
	}

	out := make(chan int64, 2000)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for instanceId := range out {
			bonus_pgcr.SendFetchMessage(outgoing, instanceId)
		}
		wg.Done()
	}()

	for _, character := range stats.Characters {
		bungie.GetActivityHistory(membershipType, request.MembershipId, character.CharacterId, out)
	}

	close(out)
	wg.Wait()
}
