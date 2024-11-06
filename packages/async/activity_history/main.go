package activity_history

import (
	"context"
	"encoding/json"
	"raidhub/packages/async"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ActivityHistoryRequest struct {
	MembershipId int64 `json:"membershipId,string"`
}

const queueName = "activity_history"

func Create() async.QueueWorker {
	create_outbound_channel()
	return async.QueueWorker{
		QueueName: queueName,
		Processer: process_request,
	}
}

func SendMessage(ch *amqp.Channel, membershipId int64) error {
	body, err := json.Marshal(ActivityHistoryRequest{
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
