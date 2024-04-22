package activity_history

import (
	"context"
	"encoding/json"
	"raidhub/shared/async"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ActivityHistoryRequest struct {
	MembershipId string `json:"membershipId"`
}

const queueName = "activity_history"

func Create() async.QueueWorker {
	return async.QueueWorker{
		QueueName: queueName,
		Processer: process_queue,
	}
}

func SendMessage(ch *amqp.Channel, membershipId int64) error {
	body, err := json.Marshal(ActivityHistoryRequest{
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
