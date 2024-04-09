package player_crawl

import (
	"context"
	"encoding/json"
	"raidhub/shared/async"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PlayerRequest struct {
	MembershipId string `json:"membershipId"`
}

const queueName = "player_requests"

func Register(numWorkers int) {
	async.RegisterQueueWorker(queueName, numWorkers, process_queue)
}

func SendPlayerCrawlMessage(ch *amqp.Channel, membershipId int64) error {
	body, err := json.Marshal(PlayerRequest{
		MembershipId: strconv.FormatInt(membershipId, 10),
	})
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		"",        // exchange
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
