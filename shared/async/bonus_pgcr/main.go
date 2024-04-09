package bonus_pgcr

import (
	"context"
	"encoding/json"
	"raidhub/shared/async"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PGCRRequest struct {
	InstanceId string `json:"instanceId"`
}

const queueName = "bonus_pgcr"

func Register(numWorkers int) {
	async.RegisterQueueWorker(queueName, numWorkers, process_queue)
}

func SendBonusPGCRMessage(ch *amqp.Channel, instanceId int64) error {
	body, err := json.Marshal(PGCRRequest{
		InstanceId: strconv.FormatInt(instanceId, 10),
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
