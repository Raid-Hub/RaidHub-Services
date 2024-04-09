package activity_history

import (
	"context"
	"encoding/json"
	"raidhub/shared/async"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ActivityHistoryRequest struct {
	MembershipId   string `json:"membershipId"`
	MembershipType int    `json:"membershipType"`
}

const queueName = "activity_history"

func Register(numWorkers int) {
	async.RegisterQueueWorker(queueName, numWorkers, process_queue)
}

func SendActivityHistoryRequest(ch *amqp.Channel, membershipType int, membershipId int64) error {
	body, err := json.Marshal(ActivityHistoryRequest{
		MembershipId:   strconv.FormatInt(membershipId, 10),
		MembershipType: membershipType,
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
