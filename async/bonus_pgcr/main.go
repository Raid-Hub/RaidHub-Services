package bonus_pgcr

import (
	"context"
	"encoding/json"
	"raidhub/async"
	"raidhub/shared/bungie"
	"raidhub/shared/pgcr"
	"strconv"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	outgoing *amqp.Channel
	once     sync.Once
)

func create_outbound_channel(conn *amqp.Connection) {
	once.Do(func() {
		outgoing, _ = conn.Channel()
	})
}

const fetchQueueName = "pgcr_fetch"

func CreateFetchWorker() async.QueueWorker {
	return async.QueueWorker{
		QueueName: fetchQueueName,
		Processer: process_fetch_queue,
	}
}

func SendFetchMessage(ch *amqp.Channel, instanceId int64) error {
	body, err := json.Marshal(PGCRFetchRequest{
		InstanceId: strconv.FormatInt(instanceId, 10),
	})
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		"",             // exchange
		fetchQueueName, // routing key (queue name)
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

const storeQueueName = "pgcr_store"

func CreateStoreWorker() async.QueueWorker {
	return async.QueueWorker{
		QueueName: storeQueueName,
		Processer: process_store_queue,
	}
}

func sendStoreMessage(ch *amqp.Channel, activity *pgcr.ProcessedActivity, raw *bungie.DestinyPostGameCarnageReport) error {
	body, err := json.Marshal(PGCRStoreRequest{
		Activity: activity,
		Raw:      raw,
	})
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		"",             // exchange
		storeQueueName, // routing key (queue name)
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
