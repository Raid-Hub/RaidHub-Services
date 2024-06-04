package player_crawl

import (
	"context"
	"encoding/json"
	"raidhub/async"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PlayerRequest struct {
	MembershipId int64 `json:"membershipId,string"`
}

const queueName = "player_requests"

func Create() async.QueueWorker {
	return async.QueueWorker{
		QueueName: queueName,
		Processer: process_queue,
	}
}

func SendMessage(ch *amqp.Channel, membershipId int64) error {
	body, err := json.Marshal(PlayerRequest{
		MembershipId: membershipId,
	})
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		"",        // exchange
		queueName, // routing key (queue name)
		true,      // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
