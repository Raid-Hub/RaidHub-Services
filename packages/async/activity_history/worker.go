package activity_history

import (
	"encoding/json"
	"log"
	"raidhub/packages/async"
	"raidhub/packages/async/bonus_pgcr"
	"raidhub/packages/bungie"
	"raidhub/packages/rabbit"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	outgoing *amqp.Channel
	once     sync.Once
)

func create_outbound_channel() {
	once.Do(func() {
		conn, err := rabbit.Init()
		if err != nil {
			log.Fatalf("Failed to create outbound channel: %s", err)
		}
		outgoing, _ = conn.Channel()
	})
}

func process_request(qw *async.QueueWorker, msg amqp.Delivery) {
	qw.Wg.Wait()
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
	if err != nil {
		log.Printf("Failed to get linked profiles: %s", err)
		return
	}

	var membershipType int
	for _, profile := range profiles {
		if profile.MembershipId == request.MembershipId {
			membershipType = profile.MembershipType
			break
		}
	}

	if membershipType == 0 {
		log.Printf("Failed to find membership type for %d", request.MembershipId)
		return
	}

	stats, err := bungie.GetHistoricalStats(membershipType, request.MembershipId)
	if err != nil {
		log.Printf("Failed to get stats: %s", err)
		return
	}

	out := make(chan int64, 2000)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for instanceId := range out {
			bonus_pgcr.SendFetchMessage(outgoing, instanceId)
		}
	}()

	var success = false
	for _, character := range stats.Characters {
		err := bungie.GetActivityHistory(membershipType, request.MembershipId, character.CharacterId, 3, out)
		if err != nil {
			log.Println(err)
			break
		}
		success = true
	}

	if success {
		log.Printf("Updating player %d history_last_crawled", request.MembershipId)
		_, err := qw.Db.Exec(`UPDATE player SET history_last_crawled = NOW() WHERE membership_id = $1`, request.MembershipId)
		if err != nil {
			log.Fatal(err)
		}
	}

	close(out)
	wg.Wait()
}
